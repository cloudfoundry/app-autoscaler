package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.jdbc.DataSourceProperties;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit.jupiter.SpringExtension;

@ExtendWith(SpringExtension.class)
@EnableConfigurationProperties(value = DataSourceProperties.class)
@SpringBootTest
public class DataSourceConfigTest {

  @Qualifier("primary")
  @Autowired
  private DataSourceProperties properties;

  @Qualifier("policy")
  @Autowired
  private DataSourceProperties policyDbProperties;

  @Test
  void datasourceCredentialsHasStringType() {
    boolean isUsernameInStringFormat = properties.getUsername() instanceof String;
    assertFalse(!isUsernameInStringFormat);

    assertInstanceOf(
        String.class, properties.getUsername(), "datasource db username should be of type string");
    assertInstanceOf(
        String.class, properties.getPassword(), "datasource db password should be of type string");
  }

  @Test
  void policyDbCredentialsHasStringType() {
    boolean isUsernameInStringFormat = policyDbProperties.getUsername() instanceof String;
    assertFalse(!isUsernameInStringFormat);

    assertInstanceOf(
        String.class,
        policyDbProperties.getUsername(),
        "policy db username should be of type string");
    assertInstanceOf(
        String.class, policyDbProperties.getPassword(), "db password should be of type string");
  }
}
