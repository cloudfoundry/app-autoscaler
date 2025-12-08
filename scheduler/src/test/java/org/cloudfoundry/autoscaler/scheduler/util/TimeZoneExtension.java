package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.TimeZone;

import org.junit.jupiter.api.extension.AfterEachCallback;
import org.junit.jupiter.api.extension.BeforeEachCallback;
import org.junit.jupiter.api.extension.Extension;
import org.junit.jupiter.api.extension.ExtensionContext;
import org.junit.jupiter.api.extension.TestTemplateInvocationContext;
import org.junit.jupiter.api.extension.TestTemplateInvocationContextProvider;

import java.util.List;
import java.util.stream.Stream;

/** Extension for running test templates with different Time zones */
public class TimeZoneExtension implements TestTemplateInvocationContextProvider {
  private static final String[] DEFAULT_ZONE_IDS = {"GMT", "America/Montreal", "Australia/Sydney"};

  @Override
  public boolean supportsTestTemplate(ExtensionContext context) {
    return true;
  }

  @Override
  public Stream<TestTemplateInvocationContext> provideTestTemplateInvocationContexts(
      ExtensionContext context) {
    return Stream.of(DEFAULT_ZONE_IDS)
        .map(
            zoneId ->
                new TestTemplateInvocationContext() {
                  @Override
                  public String getDisplayName(int invocationIndex) {
                    return "TimeZone: " + zoneId;
                  }

                  @Override
                  public List<Extension> getAdditionalExtensions() {
                    return List.of(new TimeZoneParameterResolver(zoneId));
                  }
                });
  }

  private static class TimeZoneParameterResolver
      implements BeforeEachCallback,
          AfterEachCallback {
    private final String zoneId;
    private TimeZone originalTimeZone;

    public TimeZoneParameterResolver(String zoneId) {
      this.zoneId = zoneId;
    }

    @Override
    public void beforeEach(ExtensionContext context) {
      originalTimeZone = TimeZone.getDefault();
      TimeZone.setDefault(TimeZone.getTimeZone(zoneId));
    }

    @Override
    public void afterEach(ExtensionContext context) {
      TimeZone.setDefault(originalTimeZone);
    }
  }
}

