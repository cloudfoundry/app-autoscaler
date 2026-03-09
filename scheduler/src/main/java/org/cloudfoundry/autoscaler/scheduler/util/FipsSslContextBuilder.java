package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.IOException;
import java.io.InputStream;
import java.security.KeyStore;
import java.security.KeyStoreException;
import java.security.NoSuchAlgorithmException;
import java.security.cert.CertificateException;
import java.security.cert.X509Certificate;
import java.util.ArrayList;
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
 * and won't automatically fall back to system certificates. This builder merges:
 *
 * <ol>
 *   <li>System CA certificates from the JVM's default cacerts keystore
 *   <li>Custom CA certificates from application configuration
 * </ol>
 *
 * <p>This ensures that HTTPS connections work both with:
 *
 * <ul>
 *   <li>Public services using well-known CAs (e.g., Let's Encrypt)
 *   <li>Internal services using custom/self-signed CAs (e.g., CF internal routing)
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
   * Gets the default system TrustManager with JVM cacerts.
   *
   * @return X509TrustManager with system CA certificates
   */
  private static X509TrustManager getSystemTrustManager()
      throws NoSuchAlgorithmException, KeyStoreException, IOException, CertificateException {
    // Load system truststore (cacerts)
    KeyStore systemKeyStore = KeyStore.getInstance(KeyStore.getDefaultType());
    String javaHome = System.getProperty("java.home");
    String cacertsPath = javaHome + "/lib/security/cacerts";

    logger.debug("Loading system CA certificates from: {}", cacertsPath);

    try (InputStream is = java.nio.file.Files.newInputStream(java.nio.file.Paths.get(cacertsPath))) {
      systemKeyStore.load(is, "changeit".toCharArray());
    }

    TrustManagerFactory tmf = TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm());
    tmf.init(systemKeyStore);

    for (TrustManager tm : tmf.getTrustManagers()) {
      if (tm instanceof X509TrustManager) {
        X509TrustManager x509tm = (X509TrustManager) tm;
        logger.info("System truststore loaded with {} CA certificates", x509tm.getAcceptedIssuers().length);
        return x509tm;
      }
    }

    throw new IllegalStateException("No X509TrustManager found in system TrustManagerFactory");
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