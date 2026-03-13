package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.BufferedInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.KeyStore;
import java.security.KeyStoreException;
import java.security.NoSuchAlgorithmException;
import java.security.cert.CertificateException;
import java.security.cert.CertificateFactory;
import java.security.cert.X509Certificate;
import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.TrustManagerFactory;
import javax.net.ssl.X509TrustManager;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Builds FIPS-compliant SSLContext that trusts both system CAs and custom CAs.
 *
 * <p>In FIPS mode with Bouncy Castle providers, the TrustManager requires explicit trust anchors
 * and won't automatically fall back to system certificates. Additionally, the Java buildpack's
 * Container Security Provider (which normally adds BOSH trusted certificates transparently) is
 * bypassed because we replaced SunJSSE with BCJSSE. This builder explicitly loads:
 *
 * <ol>
 *   <li>JVM cacerts (standard public CAs like Let's Encrypt, DigiCert)
 *   <li>BOSH/CF trusted certificates from /etc/ssl/certs/ca-certificates.crt (CF internal CAs)
 *   <li>Custom CA certificates from application configuration
 * </ol>
 *
 * <p>This ensures that HTTPS connections work both with:
 *
 * <ul>
 *   <li>Public services using well-known CAs (validated via JVM cacerts)
 *   <li>Internal CF services using BOSH-managed CAs (validated via ca-certificates.crt)
 *   <li>Services using custom/self-signed CAs (validated via application configuration)
 * </ul>
 */
public class FipsSslContextBuilder {
  private static final Logger logger = LoggerFactory.getLogger(FipsSslContextBuilder.class);

  private FipsSslContextBuilder() {}

  /**
   * Creates an SSLContext that trusts both system and custom CA certificates.
   *
   * @param keyManagers Optional KeyManagers for client certificate authentication (mutual TLS).
   *     Pass null if client auth is not required.
   * @param customTrustManager Optional custom TrustManager with additional trusted CAs. Pass null
   *     to use only system CAs.
   * @param protocol SSL protocol (e.g., "TLSv1.2", "TLSv1.3")
   * @return FIPS-compliant SSLContext with merged trust anchors
   * @throws RuntimeException if SSLContext creation fails
   */
  public static SSLContext buildWithSystemAndCustomTrust(
      javax.net.ssl.KeyManager[] keyManagers,
      X509TrustManager customTrustManager,
      String protocol) {
    try {
      X509TrustManager systemTrustManager = getSystemTrustManager();
      X509TrustManager mergedTrustManager =
          new MergedTrustManager(systemTrustManager, customTrustManager);

      SSLContext sslContext = SSLContext.getInstance(protocol);
      sslContext.init(keyManagers, new TrustManager[] {mergedTrustManager}, null);

      logger.info(
          "Created FIPS-compliant SSLContext with {} system CAs and {} custom CAs{}",
          systemTrustManager.getAcceptedIssuers().length,
          customTrustManager != null ? customTrustManager.getAcceptedIssuers().length : 0,
          keyManagers != null ? " and client certificate" : "");

      return sslContext;
    } catch (Exception e) {
      throw new RuntimeException("Failed to create SSLContext with merged trust anchors", e);
    }
  }

  /**
   * Creates an SSLContext that trusts system CA certificates only.
   *
   * @param keyManagers Optional KeyManagers for client certificate authentication (mutual TLS).
   *     Pass null if client auth is not required.
   * @param protocol SSL protocol (e.g., "TLSv1.2", "TLSv1.3")
   * @return FIPS-compliant SSLContext with system trust anchors
   * @throws RuntimeException if SSLContext creation fails
   */
  public static SSLContext buildWithSystemTrust(
      javax.net.ssl.KeyManager[] keyManagers, String protocol) {
    try {
      X509TrustManager systemTrustManager = getSystemTrustManager();

      SSLContext sslContext = SSLContext.getInstance(protocol);
      sslContext.init(keyManagers, new TrustManager[] {systemTrustManager}, null);

      logger.info(
          "Created FIPS-compliant SSLContext with {} system CA certificates{}",
          systemTrustManager.getAcceptedIssuers().length,
          keyManagers != null ? " and client certificate" : "");

      return sslContext;
    } catch (Exception e) {
      throw new RuntimeException("Failed to create SSLContext with system trust anchors", e);
    }
  }

  /**
   * Path to BOSH/CF trusted certificates. The Java buildpack's Container Security Provider normally
   * adds these certificates via a custom TrustManagerFactory, but since we replaced SunJSSE with
   * BCJSSE, we must load them explicitly.
   *
   * @see <a
   *     href="https://github.com/cloudfoundry/java-buildpack/blob/main/docs/framework-container_security_provider.md">
   *     Container Security Provider documentation</a>
   */
  private static final String CF_TRUSTED_CA_CERTS_PATH = "/etc/ssl/certs/ca-certificates.crt";

  /**
   * Gets the default system TrustManager with JVM cacerts merged with BOSH/CF trusted certificates.
   *
   * @return X509TrustManager with system and CF CA certificates
   */
  private static X509TrustManager getSystemTrustManager()
      throws NoSuchAlgorithmException, KeyStoreException, IOException, CertificateException {
    // Load JVM cacerts keystore
    KeyStore systemKeyStore = KeyStore.getInstance(KeyStore.getDefaultType());
    String javaHome = System.getProperty("java.home");
    String cacertsPath = javaHome + "/lib/security/cacerts";

    logger.debug("Loading JVM CA certificates from: {}", cacertsPath);

    try (InputStream is = Files.newInputStream(Paths.get(cacertsPath))) {
      systemKeyStore.load(is, "changeit".toCharArray());
    }

    int jvmCertCount = systemKeyStore.size();
    logger.info("Loaded {} CA certificates from JVM cacerts", jvmCertCount);

    // Load BOSH/CF trusted certificates from /etc/ssl/certs/ca-certificates.crt
    // This is where the Java buildpack places BOSH trusted certificates including
    // the CF system CA that signs internal service certificates (e.g., scaling engine).
    int cfCertCount = loadCfTrustedCertificates(systemKeyStore);

    TrustManagerFactory tmf =
        TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm());
    tmf.init(systemKeyStore);

    for (TrustManager tm : tmf.getTrustManagers()) {
      if (tm instanceof X509TrustManager) {
        X509TrustManager x509tm = (X509TrustManager) tm;
        logger.info(
            "System truststore loaded with {} CA certificates ({} from JVM cacerts, {} from CF trusted certs)",
            x509tm.getAcceptedIssuers().length,
            jvmCertCount,
            cfCertCount);
        return x509tm;
      }
    }

    throw new IllegalStateException("No X509TrustManager found in system TrustManagerFactory");
  }

  /**
   * Loads PEM-encoded certificates from the CF/BOSH trusted certificates file into the given
   * keystore.
   *
   * @param keyStore the keystore to add certificates to
   * @return the number of certificates loaded
   */
  private static int loadCfTrustedCertificates(KeyStore keyStore) {
    Path cfCertsPath = Paths.get(CF_TRUSTED_CA_CERTS_PATH);

    if (!Files.exists(cfCertsPath)) {
      logger.info(
          "CF trusted certificates file not found at {} (not running in Cloud Foundry?), skipping",
          CF_TRUSTED_CA_CERTS_PATH);
      return 0;
    }

    try (InputStream is = new BufferedInputStream(Files.newInputStream(cfCertsPath))) {
      CertificateFactory cf = CertificateFactory.getInstance("X.509");

      @SuppressWarnings("unchecked")
      Collection<X509Certificate> certs =
          (Collection<X509Certificate>) (Collection<?>) cf.generateCertificates(is);

      int count = 0;
      for (X509Certificate cert : certs) {
        String alias = "cf-trusted-" + count;
        if (!containsCertificate(keyStore, cert)) {
          keyStore.setCertificateEntry(alias, cert);
          count++;
          logger.debug(
              "Added CF trusted certificate: {}", cert.getSubjectX500Principal().getName());
        }
      }

      logger.info(
          "Loaded {} CF/BOSH trusted certificates from {}", count, CF_TRUSTED_CA_CERTS_PATH);
      return count;
    } catch (Exception e) {
      logger.warn(
          "Failed to load CF trusted certificates from {}: {}",
          CF_TRUSTED_CA_CERTS_PATH,
          e.getMessage());
      return 0;
    }
  }

  /**
   * Checks if a keystore already contains a certificate (by comparing encoded form) to avoid
   * duplicates when merging JVM cacerts with CF trusted certs.
   */
  private static boolean containsCertificate(KeyStore keyStore, X509Certificate cert) {
    try {
      String alias = keyStore.getCertificateAlias(cert);
      return alias != null;
    } catch (KeyStoreException e) {
      return false;
    }
  }

  /**
   * TrustManager that delegates to both system and custom trust managers.
   *
   * <p>Certificate validation logic:
   *
   * <ol>
   *   <li>Try custom TrustManager first (if provided)
   *   <li>Fall back to system TrustManager if custom validation fails
   *   <li>Only throw exception if both fail
   * </ol>
   *
   * <p>This allows services to present either:
   *
   * <ul>
   *   <li>Certificates signed by custom CAs (validated by customTrustManager)
   *   <li>Certificates signed by well-known CAs (validated by systemTrustManager)
   * </ul>
   */
  private static class MergedTrustManager implements X509TrustManager {
    private final X509TrustManager systemTrustManager;
    private final X509TrustManager customTrustManager;

    public MergedTrustManager(
        X509TrustManager systemTrustManager, X509TrustManager customTrustManager) {
      this.systemTrustManager = systemTrustManager;
      this.customTrustManager = customTrustManager;
    }

    @Override
    public void checkClientTrusted(X509Certificate[] chain, String authType)
        throws CertificateException {
      // Try custom first, then fall back to system
      try {
        if (customTrustManager != null) {
          customTrustManager.checkClientTrusted(chain, authType);
          return;
        }
      } catch (CertificateException e) {
        // Fall through to system trust manager
      }

      systemTrustManager.checkClientTrusted(chain, authType);
    }

    @Override
    public void checkServerTrusted(X509Certificate[] chain, String authType)
        throws CertificateException {
      // Try custom first, then fall back to system
      CertificateException customException = null;

      try {
        if (customTrustManager != null) {
          customTrustManager.checkServerTrusted(chain, authType);
          logger.debug("Server certificate validated against custom trust manager");
          return;
        }
      } catch (CertificateException e) {
        customException = e;
        logger.debug("Custom trust manager validation failed, trying system trust manager", e);
      }

      try {
        systemTrustManager.checkServerTrusted(chain, authType);
        logger.debug("Server certificate validated against system trust manager");
      } catch (CertificateException systemException) {
        logger.error(
            "Certificate validation failed with both custom and system trust managers. "
                + "Custom error: {}, System error: {}",
            customException != null ? customException.getMessage() : "N/A",
            systemException.getMessage());
        // Throw the system exception as it's usually more informative
        throw systemException;
      }
    }

    @Override
    public X509Certificate[] getAcceptedIssuers() {
      // Merge accepted issuers from both trust managers
      List<X509Certificate> issuers = new ArrayList<>();

      if (systemTrustManager != null) {
        for (X509Certificate cert : systemTrustManager.getAcceptedIssuers()) {
          issuers.add(cert);
        }
      }

      if (customTrustManager != null) {
        for (X509Certificate cert : customTrustManager.getAcceptedIssuers()) {
          issuers.add(cert);
        }
      }

      return issuers.toArray(new X509Certificate[0]);
    }
  }
}