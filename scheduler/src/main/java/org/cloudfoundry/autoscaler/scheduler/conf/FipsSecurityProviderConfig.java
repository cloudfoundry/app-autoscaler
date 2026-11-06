package org.cloudfoundry.autoscaler.scheduler.conf;

import java.security.Provider;
import java.security.Security;

import lombok.Getter;
import lombok.extern.slf4j.Slf4j;
import org.bouncycastle.crypto.CryptoServicesRegistrar;
import org.bouncycastle.jcajce.provider.BouncyCastleFipsProvider;
import org.bouncycastle.jsse.provider.BouncyCastleJsseProvider;

@Slf4j
public class FipsSecurityProviderConfig {

  @Getter
  private static volatile boolean initialized = false;

  private FipsSecurityProviderConfig() {}

  public static synchronized void initialize() {
    if (initialized) {
      return;
    }

    log.info("Initializing FIPS 140-3 security providers");

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

    verifyApprovedOnlyMode(CryptoServicesRegistrar.isInApprovedOnlyMode());

    logRegisteredProviders();

    log.info(
        "FIPS 140-3 security providers initialized successfully. Approved-only mode is active.");
    initialized = true;
  }

  /**
   * Guards against the JVM silently running outside FIPS approved-only mode. Package-private and
   * parameterized so the failure path is testable without having to break the running JVM's global
   * crypto state.
   */
  static void verifyApprovedOnlyMode(boolean approvedOnly) {
    if (!approvedOnly) {
      throw new IllegalStateException(
          "Failed to activate FIPS approved-only mode. "
              + "The scheduler cannot start without FIPS 140-3 compliance.");
    }
  }

  private static void logRegisteredProviders() {
    Provider[] providers = Security.getProviders();
    log.info("Registered security providers ({}):", providers.length);
    for (int i = 0; i < providers.length; i++) {
      log.info("  [{}] {} v{}", i + 1, providers[i].getName(), providers[i].getVersionStr());
    }
  }

  /**
   * Reports the registered crypto providers and BC approved-only state on the calling thread.
   * This does not claim that every JVM crypto operation uses a FIPS provider.
   */
  public static boolean isFipsEnabled() {
    return Security.getProvider("BCFIPS") instanceof BouncyCastleFipsProvider
        && Security.getProvider("BCJSSE") instanceof BouncyCastleJsseProvider jsse
        && jsse.isFipsMode()
        && CryptoServicesRegistrar.isInApprovedOnlyMode();
  }
}
