package org.cloudfoundry.autoscaler.scheduler.conf;

import java.security.Provider;
import java.security.Security;
import org.bouncycastle.crypto.CryptoServicesRegistrar;
import org.bouncycastle.jcajce.provider.BouncyCastleFipsProvider;
import org.bouncycastle.jsse.provider.BouncyCastleJsseProvider;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class FipsSecurityProviderConfig {

  private static final Logger logger = LoggerFactory.getLogger(FipsSecurityProviderConfig.class);
  private static volatile boolean initialized = false;

  private FipsSecurityProviderConfig() {}

  public static synchronized void initialize() {
    if (initialized) {
      return;
    }

    logger.info("Initializing FIPS 140-3 security providers");

    // Set global FIPS approved-only mode via system property BEFORE any BC class loads.
    // This sets CryptoServicesRegistrar.isDefaultModeApprovedMode = true for ALL threads,
    // unlike setApprovedOnlyMode(true) which only affects the calling thread.
    System.setProperty("org.bouncycastle.fips.approved_only", "true");

    Security.insertProviderAt(new BouncyCastleFipsProvider(), 1);
    Security.insertProviderAt(new BouncyCastleJsseProvider("fips:BCFIPS"), 2);

    Security.removeProvider("SunJSSE");
    Security.removeProvider("SunEC");

    // Set default algorithms to BCJSSE-compatible values since Sun providers are removed.
    // Spring Boot's DefaultSslManagerBundle uses KeyManagerFactory.getDefaultAlgorithm()
    // which returns "SunX509" by default — not available without SunJSSE.
    Security.setProperty("ssl.KeyManagerFactory.algorithm", "PKIX");
    Security.setProperty("ssl.TrustManagerFactory.algorithm", "PKIX");

    if (!CryptoServicesRegistrar.isInApprovedOnlyMode()) {
      throw new IllegalStateException(
          "Failed to activate FIPS approved-only mode. "
              + "The scheduler cannot start without FIPS 140-3 compliance.");
    }

    logRegisteredProviders();

    logger.info("FIPS 140-3 security providers initialized successfully. Approved-only mode is active.");
    initialized = true;
  }

  private static void logRegisteredProviders() {
    Provider[] providers = Security.getProviders();
    logger.info("Registered security providers ({}):", providers.length);
    for (int i = 0; i < providers.length; i++) {
      logger.info("  [{}] {} v{}", i + 1, providers[i].getName(), providers[i].getVersionStr());
    }
  }

  public static boolean isInitialized() {
    return initialized;
  }
}
