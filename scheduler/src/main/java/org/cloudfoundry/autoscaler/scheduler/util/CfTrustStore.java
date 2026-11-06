package org.cloudfoundry.autoscaler.scheduler.util;

import java.nio.file.Files;
import java.nio.file.Path;
import java.security.KeyStore;
import java.security.cert.Certificate;
import java.util.Arrays;
import java.util.Collections;
import java.util.LinkedHashSet;
import java.util.Set;
import javax.net.ssl.TrustManagerFactory;
import javax.net.ssl.X509TrustManager;
import org.springframework.boot.ssl.pem.PemContent;

/** Composes JVM, CF/BOSH, and configured trust anchors for outbound HTTPS. */
public final class CfTrustStore {

  private static final Path DEFAULT_CF_CA_CERTS = Path.of("/etc/ssl/certs/ca-certificates.crt");

  private CfTrustStore() {}

  /**
   * Builds a certificate-only store without modifying the configured store.
   *
   * @param configured optional SSL bundle truststore
   * @param cfCaCerts optional CF_CA_CERTS override; the default file may be absent outside CF
   * @return combined trust anchors; client identity certificates are supplied separately
   */
  public static KeyStore compose(KeyStore configured, Path cfCaCerts) {
    try {
      TrustManagerFactory factory =
          TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm());
      // Lets JSSE honor javax.net.ssl.trustStore*, jssecacerts, and cacerts.
      factory.init((KeyStore) null);
      X509TrustManager defaults =
          Arrays.stream(factory.getTrustManagers())
              .filter(X509TrustManager.class::isInstance)
              .map(X509TrustManager.class::cast)
              .findFirst()
              .orElseThrow(() -> new IllegalStateException("No default X.509 trust manager"));
      Set<Certificate> certificates =
          new LinkedHashSet<>(Arrays.asList(defaults.getAcceptedIssuers()));

      // The buildpack's SunJSSE-based container provider is bypassed under BCJSSE.
      Path cfPath = cfCaCerts != null ? cfCaCerts : DEFAULT_CF_CA_CERTS;
      if (cfCaCerts != null || !Files.notExists(cfPath)) {
        certificates.addAll(PemContent.load(cfPath).getCertificates());
      }
      if (configured != null) {
        for (String alias : Collections.list(configured.aliases())) {
          Certificate certificate = configured.getCertificate(alias);
          if (certificate != null) {
            certificates.add(certificate);
          }
        }
      }

      KeyStore combined = KeyStore.getInstance(KeyStore.getDefaultType());
      combined.load(null, null);
      int index = 0;
      for (Certificate certificate : certificates) {
        combined.setCertificateEntry("ca-" + index++, certificate);
      }
      return combined;
    } catch (Exception e) {
      throw new IllegalStateException("Could not compose outbound HTTPS truststore", e);
    }
  }
}
