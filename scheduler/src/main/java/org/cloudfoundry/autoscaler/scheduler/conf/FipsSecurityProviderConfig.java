package org.cloudfoundry.autoscaler.scheduler.conf;

import java.security.Provider;
import java.security.Security;
import java.util.Map;
import org.bouncycastle.crypto.CryptoServicesRegistrar;
import org.bouncycastle.jcajce.provider.BouncyCastleFipsProvider;
import org.bouncycastle.jsse.provider.BouncyCastleJsseProvider;
import org.cloudfoundry.autoscaler.scheduler.util.VcapServicesHelper;
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

    logger.info(
        "FIPS 140-3 security providers initialized successfully. Approved-only mode is active.");
    initialized = true;
  }

  /**
   * Checks whether FIPS mode is enabled by reading the scheduler-config from VCAP_SERVICES.
   *
   * <p>Looks for {@code autoscaler.fips_mode: true} in the scheduler-config service credentials.
   * Falls back to {@code false} if VCAP_SERVICES is not set or the property is absent.
   */
  public static boolean isFipsModeEnabled() {
    String vcapServices = System.getenv("VCAP_SERVICES");
    return VcapServicesHelper.findServiceByTag(vcapServices, VcapServicesHelper.SCHEDULER_CONFIG_TAG)
        .map(FipsSecurityProviderConfig::extractFipsMode)
        .orElse(false);
  }

  @SuppressWarnings("unchecked")
  private static boolean extractFipsMode(Map<String, Object> service) {
    Object credentials = service.get("credentials");
    if (!(credentials instanceof Map)) {
      return false;
    }
    Map<String, Object> creds = (Map<String, Object>) credentials;

    // Check autoscaler.fips_mode path
    Object autoscaler = creds.get("autoscaler");
    if (autoscaler instanceof Map) {
      Object fipsMode = ((Map<String, Object>) autoscaler).get("fips_mode");
      return Boolean.TRUE.equals(fipsMode);
    }

    return false;
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
