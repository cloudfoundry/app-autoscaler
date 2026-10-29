package org.cloudfoundry.autoscaler.scheduler.filter;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatCode;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.GeneralSecurityException;
import java.security.cert.CertificateFactory;
import java.security.cert.X509Certificate;
import java.time.Clock;
import java.time.Instant;
import java.time.ZoneId;
import java.time.ZoneOffset;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.Executors;
import org.cloudfoundry.autoscaler.scheduler.conf.CfHttpConfiguration;
import org.cloudfoundry.autoscaler.scheduler.conf.CfServerConfiguration;
import org.cloudfoundry.autoscaler.scheduler.conf.CfServerConfiguration.VerificationMode;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;
import org.slf4j.LoggerFactory;
import org.springframework.mock.env.MockEnvironment;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.mock.web.MockHttpServletResponse;

/** Uses real PKIX paths; the FIPS subclass runs the same cases in a separate approved-only JVM. */
class XfccChainVerifierTest {
  private static final Path CERTS = Path.of("src/test/resources/certs");

  @TempDir Path directory;

  private CfServerConfiguration config;
  private MockEnvironment environment;
  private MutableClock clock;
  private Path roots;
  private Path instance;
  private X509Certificate leaf;

  @BeforeEach
  void setUp() throws Exception {
    roots = Files.createDirectory(directory.resolve("roots"));
    instance = directory.resolve("instance.crt");
    Files.copy(CERTS.resolve("test-ca.crt"), roots.resolve("root.pem"));
    writeChain("xfcc-valid.crt", "xfcc-issuer.crt");
    config = new CfServerConfiguration();
    config.setSystemCertPath(roots.toString());
    config.setInstanceCertPath(instance.toString());
    config.setValidOrgGuid("validorg");
    config.setValidSpaceGuid("validspace");
    config.setHealthserver(new CfServerConfiguration.HealthServer());
    environment = new MockEnvironment();
    clock = new MutableClock();
    leaf = certificate("xfcc-valid.crt");
  }

  @Test
  void verifiesTheContainerSuppliedThreeLevelChain() {
    assertThatCode(() -> verifier().verify(leaf)).doesNotThrowAnyException();
  }

  @ParameterizedTest
  @ValueSource(strings = {"xfcc-no-eku.crt", "xfcc-any-eku.crt"})
  void acceptsAbsentOrAnyPurposeEku(String name) throws Exception {
    X509Certificate caller = certificate(name);
    assertThatCode(() -> verifier().verify(caller)).doesNotThrowAnyException();
  }

  @ParameterizedTest
  @ValueSource(strings = {"xfcc-server-eku.crt", "xfcc-critical.crt", "test-ca.crt"})
  void rejectsWrongPurposeUnknownCriticalExtensionAndCaCallers(String name) throws Exception {
    X509Certificate caller = certificate(name);
    assertRejected(verifier(), caller);
  }

  @Test
  void rejectsExpiredCertificates() {
    clock.advanceSeconds(370L * 24 * 3600);
    assertRejected(verifier(), leaf);
  }

  @Test
  void rejectsAnIntermediateRestrictedToServerAuthentication() throws Exception {
    writeChain("xfcc-valid.crt", "xfcc-server-issuer.crt");
    assertRejected(verifier(), leaf);
  }

  @Test
  void rejectsCertificatesNotYetValid() {
    clock.advanceSeconds(-24 * 3600);
    assertRejected(verifier(), leaf);
  }

  @Test
  void rejectsAnUnverifiableSignature() throws Exception {
    X509Certificate unrelated = certificate("test-scheduler.crt");
    // This has a valid root signature, so change its encoded signature bytes
    // without changing the identity/validity being parsed.
    byte[] encoded = unrelated.getEncoded();
    encoded[encoded.length - 1] ^= 1;
    X509Certificate tampered =
        (X509Certificate)
            CertificateFactory.getInstance("X.509")
                .generateCertificate(new java.io.ByteArrayInputStream(encoded));
    assertRejected(verifier(), tampered);
  }

  @ParameterizedTest
  @ValueSource(strings = {"empty", "leaf-only", "non-CA", "malformed", "missing"})
  void rejectsUnavailableOrInvalidMountedChains(String shape) throws Exception {
    switch (shape) {
      case "empty" -> Files.writeString(instance, "");
      case "leaf-only" -> writeChain("xfcc-valid.crt");
      case "non-CA" -> writeChain("xfcc-valid.crt", "test-scheduler.crt");
      case "malformed" ->
          Files.writeString(
              instance, "-----BEGIN CERTIFICATE-----\nbad\n-----END CERTIFICATE-----");
      case "missing" -> Files.delete(instance);
      default -> throw new IllegalArgumentException(shape);
    }
    assertRejected(verifier(), leaf);
  }

  @ParameterizedTest
  @ValueSource(strings = {"empty", "non-CA", "malformed", "missing"})
  void rejectsUnavailableOrInvalidRootStores(String shape) throws Exception {
    Files.delete(roots.resolve("root.pem"));
    switch (shape) {
      case "empty" -> assertThat(roots).isEmptyDirectory();
      case "non-CA" -> Files.copy(CERTS.resolve("xfcc-valid.crt"), roots.resolve("leaf.pem"));
      case "malformed" -> Files.writeString(roots.resolve("bad.pem"), "not a certificate");
      case "missing" -> Files.delete(roots);
      default -> throw new IllegalArgumentException(shape);
    }
    assertRejected(verifier(), leaf);
  }

  @Test
  void readsAllRootBlocksAndIgnoresDirectoriesAndNonCaCertificates() throws Exception {
    Files.createDirectory(roots.resolve("subdir"));
    Files.writeString(
        roots.resolve("bundle.pem"),
        Files.readString(CERTS.resolve("test-ca.crt"))
            + Files.readString(CERTS.resolve("xfcc-valid.crt")));
    assertThatCode(() -> verifier().verify(leaf)).doesNotThrowAnyException();
  }

  @Test
  void throttlesFailedRefreshesAndRecoversAtTheNextInterval() throws Exception {
    XfccChainVerifier verifier = verifier();
    verifier.verify(leaf);
    Files.delete(instance);
    verifier.verify(leaf);
    clock.advanceSeconds(30);
    assertRejected(verifier, leaf);
    writeChain("xfcc-valid.crt", "xfcc-issuer.crt");
    assertRejected(verifier, leaf);
    clock.advanceSeconds(30);
    assertThatCode(() -> verifier.verify(leaf)).doesNotThrowAnyException();
  }

  @Test
  void refreshesMountedChainsButNeverRereadsContainerRoots() throws Exception {
    XfccChainVerifier verifier = verifier();
    verifier.verify(leaf);
    Files.delete(roots.resolve("root.pem"));
    writeChain("xfcc-valid.crt", "test-scheduler.crt");
    verifier.verify(leaf);
    clock.advanceSeconds(30);
    assertRejected(verifier, leaf);
    writeChain("xfcc-valid.crt", "xfcc-issuer.crt");
    clock.advanceSeconds(30);
    verifier.verify(leaf);
    assertRejected(verifier(), leaf);
  }

  @Test
  void supportsConcurrentRequests() throws Exception {
    XfccChainVerifier verifier = verifier();
    var executor = Executors.newFixedThreadPool(4);
    try {
      List<java.util.concurrent.Future<?>> requests = new ArrayList<>();
      for (int i = 0; i < 20; i++) {
        requests.add(
            executor.submit(
                () -> {
                  assertThatCode(() -> verifier.verify(leaf)).doesNotThrowAnyException();
                }));
      }
      for (var request : requests) {
        request.get();
      }
    } finally {
      executor.shutdownNow();
    }
  }

  @Test
  void shadowModeVerifiesButThrottlesFailureWarnings() throws Exception {
    config.setXfccVerificationMode(VerificationMode.SHADOW);
    Files.delete(instance);
    Logger logger = (Logger) LoggerFactory.getLogger(XfccChainVerifier.class);
    ListAppender<ILoggingEvent> appender = new ListAppender<>();
    appender.start();
    logger.addAppender(appender);
    try {
      XfccChainVerifier verifier = verifier();
      verifier.verify(leaf);
      verifier.verify(leaf);
      assertThat(appender.list).hasSize(1);
      clock.advanceSeconds(30);
      verifier.verify(leaf);
      assertThat(appender.list).hasSize(2);
    } finally {
      logger.detachAppender(appender);
      appender.stop();
    }
  }

  @Test
  void offModeDoesNotRequireTrustFilesOrCertificate() {
    config.setXfccVerificationMode(VerificationMode.OFF);
    config.setSystemCertPath(null);
    config.setInstanceCertPath("");
    assertThatCode(() -> verifier().verify(null)).doesNotThrowAnyException();
  }

  @Test
  void environmentPathsAreUsedUnlessExplicitlyConfigured() throws Exception {
    environment.withProperty("CF_SYSTEM_CERT_PATH", roots.toString());
    environment.withProperty("CF_INSTANCE_CERT", instance.toString());
    config.setSystemCertPath(null);
    config.setInstanceCertPath("");
    new XfccChainVerifier(config, environment).verify(leaf);
    config.setSystemCertPath(roots.toString());
    config.setInstanceCertPath(instance.toString());
    environment.withProperty("CF_SYSTEM_CERT_PATH", "missing");
    environment.withProperty("CF_INSTANCE_CERT", "missing");
    assertThatCode(() -> verifier().verify(leaf)).doesNotThrowAnyException();
  }

  @Test
  void filterRejectsChainFailuresBeforeCheckingIdentity() throws Exception {
    CfHttpConfiguration http = new CfHttpConfiguration();
    http.setPort(8080);
    Files.delete(instance);
    HttpAuthFilter filter = new HttpAuthFilter(config, http, verifier());
    MockHttpServletRequest request = new MockHttpServletRequest();
    request.setLocalPort(8080);
    request.setRequestURI("/v1/apps/app/schedules");
    request.addHeader("X-Forwarded-Client-Cert", Files.readString(CERTS.resolve("xfcc-valid.crt")));
    MockHttpServletResponse response = new MockHttpServletResponse();
    filter.doFilterInternal(
        request,
        response,
        (req, res) -> {
          throw new AssertionError("Untrusted request reached the filter chain");
        });
    assertThat(response.getStatus()).isEqualTo(401);
  }

  private XfccChainVerifier verifier() {
    return new XfccChainVerifier(config, environment, clock);
  }

  private static void assertRejected(XfccChainVerifier verifier, X509Certificate certificate) {
    assertThatThrownBy(() -> verifier.verify(certificate))
        .isInstanceOf(GeneralSecurityException.class)
        .hasMessage("XFCC certificate chain verification failed");
  }

  private static X509Certificate certificate(String name) throws Exception {
    try (InputStream input = Files.newInputStream(CERTS.resolve(name))) {
      return (X509Certificate) CertificateFactory.getInstance("X.509").generateCertificate(input);
    }
  }

  private void writeChain(String... names) throws Exception {
    StringBuilder pem = new StringBuilder();
    for (String name : names) {
      pem.append(Files.readString(CERTS.resolve(name)));
    }
    Files.writeString(instance, pem);
  }

  private static class MutableClock extends Clock {
    private Instant now = Instant.now();

    void advanceSeconds(long seconds) {
      now = now.plusSeconds(seconds);
    }

    @Override
    public ZoneId getZone() {
      return ZoneOffset.UTC;
    }

    @Override
    public Clock withZone(ZoneId zone) {
      return this;
    }

    @Override
    public Instant instant() {
      return now;
    }
  }
}
