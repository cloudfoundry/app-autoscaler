package org.cloudfoundry.autoscaler.scheduler.filter;

import static org.assertj.core.api.AssertionsForClassTypes.assertThat;

import jakarta.servlet.FilterChain;
import org.junit.Before;
import org.junit.Test;
import org.springframework.mock.web.MockFilterChain;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.mock.web.MockHttpServletResponse;

public class HttpAuthFilterTest {

  private MockHttpServletRequest request;
  private MockHttpServletResponse response;

  private FilterChain filterChain;
  private HttpAuthFilter httpAuthFilter;

  @Before
  public void setup() {
    request = new MockHttpServletRequest();
    response = new MockHttpServletResponse();
    filterChain = new MockFilterChain();
    httpAuthFilter = new HttpAuthFilter();
  }

  @Test
  public void testDoFilterHttpsRequestShouldNotConsiderXfccAndReturnSuccess() throws Exception {

    this.request.setScheme("https");

    httpAuthFilter.doFilterInternal(request, response, filterChain);

    assertThat(response.getStatus()).isEqualTo(200);
    assertThat(request.getRequestURL().toString()).isEqualTo("https://localhost:80");
  }

  @Test
  public void testDoFilterWithMissingXfccHeaderGetsSkipped() throws Exception {

    httpAuthFilter.doFilterInternal(request, response, filterChain);
    assertThat(response.getStatus()).isEqualTo(200);
  }

  @Test
  public void testDoFilterWithXfccHeaderReturnUnauthorized() throws Exception {

    String certValue =
        "MIIDczCCAlugAwIBAgIRAPAw7qCDM8L3MCWZqx+mS8owDQYJKoZIhvcNAQELBQAwZTEYMBYGA1UEChMPTXkgT3JnYW5pemF0aW9uMTUwMwYDVQQLEyxPVT1zcGFjZTp2YWxpZHNwYWNlK09VPW9yZ2FuaXphdGlvbjp2YWxpZG9yZzESMBAGA1UEAxMJbG9jYWxob3N0MB4XDTI1MDMwMzEwMDQ0OVoXDTI2MDMwMzEwMDQ0OVowZTEYMBYGA1UEChMPTXkgT3JnYW5pemF0aW9uMTUwMwYDVQQLEyxPVT1zcGFjZTp2YWxpZHNwYWNlK09VPW9yZ2FuaXphdGlvbjp2YWxpZG9yZzESMBAGA1UEAxMJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2NByZp2b1JEuM/x+M6IcHx01lF5V1gFoaYi6qMjd77gqgVFQDHIusp37gswiO0vt3DiMYnVgw9Nc8af4ZG5Y+XfSlMUHkQzj/IkCzbMJNHT0Ij4u5XlFYMUg53AK/MADyNqB6tv0105wcro32h7mmkTYEBG+XCwkr+x1zihqvzJXHa3u0V3LIzKEp+c/Rf7R9UMHH11gm6m34lYPgQVOr+FkA2QCGuJyOzGb/g7R7Xzm8VMjf4ZceSPsrLYLVr8xFIA+/vJ5iJGKkHL75yd/+A7A1x1KufSXfTQL5IKKQWynJyAPBFRglyyDZUt2L+HbRDA4DpYdV90Ou5f716S+mwIDAQABox4wHDAaBgNVHREEEzARgglsb2NhbGhvc3SHBH8AAAEwDQYJKoZIhvcNAQELBQADggEBAHI9YEzyvh1XVcb1jVJaoryhmK90NAWeFq3eJkRvcsXFPfNDvE4flXcuAsfGXletxQvhgWwX6ypq2G0EwSUkdZX2yPQ67OJoOyTTduWUmkOMkS7IpnucRJE6Js+x+csNzrpHEk76h383i7YuRu33vX8IJppyeWRfYskGayvszqJjymJhi6LfBLgkYZxtAnsvr1GPSQ/X1nv27rJEpjcGezvqUFwMEP3rb0377z1kq3IBDZe4yTB/DgnYhVNsy6ZOZGIXAPx6zPAYN9pTAhcuw7DM3zNAMAS51qAKbI3BeEomNFSZG5b6hLojdxVxneNPnDyihOAYR3V8kvjYKdgbfQA=";

    request.addHeader("X-Forwarded-Client-Cert", certValue);

    httpAuthFilter.doFilterInternal(request, response, filterChain);

    assertThat(response.getStatus()).isEqualTo(403);
    assertThat(response.getErrorMessage()).isEqualTo("Unauthorized OrganizationalUnit");
  }

  @Test
  public void testDoFilterWithXfccHeaderReturnSuccess() throws Exception {
    String certValue =
        "MIIDczCCAlugAwIBAgIRAPAw7qCDM8L3MCWZqx+mS8owDQYJKoZIhvcNAQELBQAwZTEYMBYGA1UEChMPTXkgT3JnYW5pemF0aW9uMTUwMwYDVQQLEyxPVT1zcGFjZTp2YWxpZHNwYWNlK09VPW9yZ2FuaXphdGlvbjp2YWxpZG9yZzESMBAGA1UEAxMJbG9jYWxob3N0MB4XDTI1MDMwMzEwMDQ0OVoXDTI2MDMwMzEwMDQ0OVowZTEYMBYGA1UEChMPTXkgT3JnYW5pemF0aW9uMTUwMwYDVQQLEyxPVT1zcGFjZTp2YWxpZHNwYWNlK09VPW9yZ2FuaXphdGlvbjp2YWxpZG9yZzESMBAGA1UEAxMJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2NByZp2b1JEuM/x+M6IcHx01lF5V1gFoaYi6qMjd77gqgVFQDHIusp37gswiO0vt3DiMYnVgw9Nc8af4ZG5Y+XfSlMUHkQzj/IkCzbMJNHT0Ij4u5XlFYMUg53AK/MADyNqB6tv0105wcro32h7mmkTYEBG+XCwkr+x1zihqvzJXHa3u0V3LIzKEp+c/Rf7R9UMHH11gm6m34lYPgQVOr+FkA2QCGuJyOzGb/g7R7Xzm8VMjf4ZceSPsrLYLVr8xFIA+/vJ5iJGKkHL75yd/+A7A1x1KufSXfTQL5IKKQWynJyAPBFRglyyDZUt2L+HbRDA4DpYdV90Ou5f716S+mwIDAQABox4wHDAaBgNVHREEEzARgglsb2NhbGhvc3SHBH8AAAEwDQYJKoZIhvcNAQELBQADggEBAHI9YEzyvh1XVcb1jVJaoryhmK90NAWeFq3eJkRvcsXFPfNDvE4flXcuAsfGXletxQvhgWwX6ypq2G0EwSUkdZX2yPQ67OJoOyTTduWUmkOMkS7IpnucRJE6Js+x+csNzrpHEk76h383i7YuRu33vX8IJppyeWRfYskGayvszqJjymJhi6LfBLgkYZxtAnsvr1GPSQ/X1nv27rJEpjcGezvqUFwMEP3rb0377z1kq3IBDZe4yTB/DgnYhVNsy6ZOZGIXAPx6zPAYN9pTAhcuw7DM3zNAMAS51qAKbI3BeEomNFSZG5b6hLojdxVxneNPnDyihOAYR3V8kvjYKdgbfQA=";

    request.addHeader("X-Forwarded-Client-Cert", certValue);

    httpAuthFilter.setValidOrgGuid("validorg");
    httpAuthFilter.setValidSpaceGuid("validspace");
    httpAuthFilter.doFilterInternal(request, response, filterChain);

    assertThat(response.getStatus()).isEqualTo(200);
  }

  @Test
  public void testDoFilterWithInvalidCertificateThrowsException() throws Exception {
    String certValue =
        "INVALIDMIIEPzCCAiegAwIBAgIRAJMGSSBnk/bFWozdSP+XA+swDQYJKoZIhvcNAQELBQAwFTETMBEGA1UEAxMKZ29yb3V0ZXJDQTAeFw0yNTAyMTgxMTM5MTBaFw0yNjAyMTgxMTM5MTBaMGAxGDAWBgNVBAoTD015IE9yZ2FuaXphdGlvbjEwMC4GA1UECxMnc3BhY2U6c29tZS1zcGFjZS1ndWlkIG9yZzpzb21lLW9yZy1ndWlkMRIwEAYDVQQDEwlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC2eIWURdSNflhueDJFaQ3Wwwq4rnEMxpPxat8YmIfP1nbkAZsjoe5PhsDVs8aTF5d4NSpBwglFP7HgyJe3TBJrSl96mRxq53CZr6Ye80vUY7dZEmcUbUJqxkHDaeIPozi1xwOeczq0hh2szKTi6N9R61d9K8c8lVagVGDZexbvDbiTruppB1q8ZeRFo9htzaLBBIZfCJ7mhtpwLshYZfwnpQBMu52wXYiFJtpFG8aURVIt7MIt3hlotwmybqEjqZfws8sgwGk1StCwN6IRUkQaT47xtJwtICfzbeOth88Zz056Q/rW4Lm62p7jtwL3c1EBXJqYjZzx1GJRb7tRrQ3VAgMBAAGjPzA9MB8GA1UdIwQYMBaAFJ8B6mcRXvcab7FZnuROUGoakOoUMBoGA1UdEQQTMBGCCWxvY2FsaG9zdIcEfwAAATANBgkqhkiG9w0BAQsFAAOCAgEAUMR8NvlO/komws2FcIo5FGBmEyoUbsqNTntXh95kJHBY3OKhmvUXHC+301aaTmeFQp/sUtQL4Wc2AFioiaJYgf8przSiwgtZXfbJxhqV0c11xTNu5xgnwwNdKLXz/OTXhbcomzjVPDPFTtvgqKncRLdNtOGzb6XhkHm5quP2CL64IsGiRPbiDZtFuCBBF9lPPT+sHUWbyMvmSExUHHaH+ZH54nz0InraV5n9sUYoXh52m/B0Ou1Kij2JMCudPHxRvDaO9Tg/q1Fxz8rea/y5+sBl2WZB6sXcWegnhzuG7VCM+u4oKKHToU/VZCPz6FlCAPWrSVBh66oBNdFI5znY0Z2SkKOTKkuch28cadR4RB+eHsMdEoquY+meCbX+WJxICIi7nyiClCgxHaL2UXALe0pjyCr69+Eg3s9fxklAMtU+XH70Ibz9nzMsTbN6zyAeXqbQs6pJpd3tprk/OgcKjDO3I6/kBhAj9bEMXmYL09ImZTWLx/IqWnsj4OooUrYJ00AeQwNO4J5BqCD6mkl92kkT5OLcp512mNAxNnD9LkRIW/dU8u1NgP0GoYRC7rB7lHpmAl5eztwqkkafJcFildmALjeBxJi/QSXHIFVIX5s31QNHrkxSxFBH/glMWkfaI7acADYbfmE3ftkevId4b/NtTJoxUoL8GDmnnHbEN90";
    request.addHeader("X-Forwarded-Client-Cert", certValue);

    httpAuthFilter.setValidOrgGuid("some-org-guid");
    httpAuthFilter.setValidSpaceGuid("some-space-guid");
    httpAuthFilter.doFilterInternal(request, response, filterChain);

    assertThat(response.getStatus()).isEqualTo(400);
    assertThat(response.getErrorMessage())
        .contains("Invalid certificate: Could not parse certificate");
  }

  @Test
  public void testDoFilterHttpHealthReturnsSuccess() throws Exception {
    var username = "test-user";
    var password = "test-password";

    this.request.setScheme("http");
    this.request.setRequestURI("/health");
    this.request.setMethod("GET");
    this.request.addHeader(
        "Authorization",
        "Basic "
            + java.util.Base64.getEncoder().encodeToString((username + ":" + password).getBytes()));

    httpAuthFilter.setHealthServerUsername(username);
    httpAuthFilter.setHealthServerPassword(password);
    httpAuthFilter.doFilterInternal(request, response, filterChain);
    assertThat(response.getStatus()).isEqualTo(200);
  }

  @Test
  public void testDoFilterHttpHealthReturnsErrorWithWrongCredentials() throws Exception {
    var username = "test-username";
    var password = "test-password";

    this.request.setScheme("http");
    this.request.setRequestURI("/health");
    this.request.setMethod("GET");
    this.request.addHeader(
        "Authorization",
        "Basic "
            + java.util.Base64.getEncoder().encodeToString((username + ":" + password).getBytes()));

    httpAuthFilter.setHealthServerUsername("correct-user");
    httpAuthFilter.setHealthServerPassword("correct-password");
    httpAuthFilter.doFilterInternal(request, response, filterChain);
    assertThat(response.getStatus()).isEqualTo(401);
    assertThat(response.getErrorMessage()).isEqualTo("Unauthorized");
  }
}
