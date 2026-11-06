package org.cloudfoundry.autoscaler.scheduler.filter;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.GeneralSecurityException;
import java.security.cert.CertPathBuilder;
import java.security.cert.CertStore;
import java.security.cert.CertificateFactory;
import java.security.cert.CollectionCertStoreParameters;
import java.security.cert.PKIXBuilderParameters;
import java.security.cert.PKIXCertPathBuilderResult;
import java.security.cert.TrustAnchor;
import java.security.cert.X509CertSelector;
import java.security.cert.X509Certificate;
import java.time.Clock;
import java.time.Instant;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import lombok.extern.slf4j.Slf4j;
import org.cloudfoundry.autoscaler.scheduler.conf.CfServerConfiguration;
import org.cloudfoundry.autoscaler.scheduler.conf.CfServerConfiguration.VerificationMode;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.env.Environment;
import org.springframework.stereotype.Component;

/** Container roots are immutable; only the mounted instance certificate chain is refreshed. */
@Component
@Slf4j
public class XfccChainVerifier {
  private static final long REFRESH_SECONDS = 30;
  private static final String CLIENT_AUTH = "1.3.6.1.5.5.7.3.2";
  private static final String ANY_USAGE = "2.5.29.37.0";

  private final VerificationMode mode;
  private final Path instancePath;
  private final Set<TrustAnchor> roots = new HashSet<>();
  private final Clock clock;
  private Exception rootError;
  private Exception refreshError;
  private List<X509Certificate> intermediates = List.of();
  private Instant nextRefresh = Instant.MIN;
  private Instant nextWarning = Instant.MIN;

  @Autowired
  public XfccChainVerifier(CfServerConfiguration config, Environment environment) {
    this(config, environment, Clock.systemUTC());
  }

  XfccChainVerifier(CfServerConfiguration config, Environment environment, Clock clock) {
    this.mode = config.getXfccVerificationMode();
    this.clock = clock;
    this.instancePath =
        Path.of(
            path(
                config.getInstanceCertPath(),
                environment,
                "CF_INSTANCE_CERT",
                "/etc/cf-instance-credentials/instance.crt"));
    if (mode != VerificationMode.OFF) {
      try (var files =
          Files.list(
              Path.of(
                  path(
                      config.getSystemCertPath(),
                      environment,
                      "CF_SYSTEM_CERT_PATH",
                      "/etc/cf-system-certificates")))) {
        for (Path file : files.filter(Files::isRegularFile).toList()) {
          for (X509Certificate certificate : readCertificates(file)) {
            if (certificate.getBasicConstraints() >= 0) {
              roots.add(new TrustAnchor(certificate, null));
            }
          }
        }
        if (roots.isEmpty()) {
          throw new GeneralSecurityException("No XFCC trust anchors");
        }
      } catch (IOException | GeneralSecurityException e) {
        rootError = e;
      }
    }
  }

  private static String path(
      String configured, Environment environment, String name, String fallback) {
    return configured != null && !configured.isEmpty()
        ? configured
        : environment.getProperty(name, fallback);
  }

  private static List<X509Certificate> readCertificates(Path path)
      throws IOException, GeneralSecurityException {
    try (InputStream input = Files.newInputStream(path)) {
      return CertificateFactory.getInstance("X.509").generateCertificates(input).stream()
          .map(X509Certificate.class::cast)
          .toList();
    }
  }

  /** No certificate/subject data is included in warnings or error responses. */
  public void verify(X509Certificate leaf) throws GeneralSecurityException {
    if (mode == VerificationMode.OFF) {
      return;
    }
    Instant now = clock.instant();
    try {
      List<X509Certificate> candidates = snapshot(now);
      if (leaf.getBasicConstraints() >= 0) {
        throw new GeneralSecurityException("XFCC caller certificate is a CA");
      }
      checkClientUsage(leaf);
      X509CertSelector target = new X509CertSelector();
      target.setCertificate(leaf);
      PKIXBuilderParameters parameters = new PKIXBuilderParameters(roots, target);
      parameters.setRevocationEnabled(false);
      // PKIX and X509 require Date; convert only at these API boundaries.
      parameters.setDate(java.util.Date.from(now));
      parameters.setMaxPathLength(-1);
      List<X509Certificate> certificates = new ArrayList<>(candidates);
      certificates.add(leaf);
      parameters.addCertStore(
          CertStore.getInstance("Collection", new CollectionCertStoreParameters(certificates)));
      PKIXCertPathBuilderResult result =
          (PKIXCertPathBuilderResult) CertPathBuilder.getInstance("PKIX").build(parameters);
      for (var certificate : result.getCertPath().getCertificates()) {
        checkClientUsage((X509Certificate) certificate);
      }
      checkClientUsage(result.getTrustAnchor().getTrustedCert());
      result.getTrustAnchor().getTrustedCert().checkValidity(java.util.Date.from(now));
      log.debug("XFCC certificate chain verified");
    } catch (GeneralSecurityException e) {
      if (mode != VerificationMode.SHADOW) {
        throw new GeneralSecurityException("XFCC certificate chain verification failed", e);
      }
      warn(now);
    }
  }

  private static void checkClientUsage(X509Certificate certificate)
      throws GeneralSecurityException {
    List<String> usage = certificate.getExtendedKeyUsage();
    if (usage != null && !usage.contains(CLIENT_AUTH) && !usage.contains(ANY_USAGE)) {
      throw new GeneralSecurityException("XFCC chain does not permit client authentication");
    }
  }

  private synchronized List<X509Certificate> snapshot(Instant now) throws GeneralSecurityException {
    if (!now.isBefore(nextRefresh)) {
      nextRefresh = now.plusSeconds(REFRESH_SECONDS);
      try {
        List<X509Certificate> chain = readCertificates(instancePath);
        if (chain.size() < 2) {
          throw new GeneralSecurityException("Instance certificate chain has no intermediate");
        }
        List<X509Certificate> issuers = chain.subList(1, chain.size());
        for (X509Certificate issuer : issuers) {
          if (issuer.getBasicConstraints() < 0) {
            throw new GeneralSecurityException(
                "Instance certificate chain contains a non-CA issuer");
          }
        }
        intermediates = List.copyOf(issuers);
        refreshError = null;
      } catch (IOException | GeneralSecurityException e) {
        refreshError = e;
      }
    }
    if (rootError != null) {
      throw new GeneralSecurityException("XFCC trust anchors unavailable", rootError);
    }
    if (refreshError != null) {
      throw new GeneralSecurityException("XFCC instance chain unavailable", refreshError);
    }
    return intermediates;
  }

  private synchronized void warn(Instant now) {
    if (!now.isBefore(nextWarning)) {
      nextWarning = now.plusSeconds(REFRESH_SECONDS);
      log.warn("XFCC chain verification failed in shadow mode");
    }
  }
}
