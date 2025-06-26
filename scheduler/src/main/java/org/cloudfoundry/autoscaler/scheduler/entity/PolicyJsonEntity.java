package org.cloudfoundry.autoscaler.scheduler.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import jakarta.validation.constraints.NotNull;
import java.sql.ResultSet;
import java.sql.SQLException;
import org.springframework.jdbc.core.RowMapper;

@Entity
@Table(name = "policy_json")
public class PolicyJsonEntity implements RowMapper<PolicyJsonEntity> {

  @Id
  @NotNull
  @Column(name = "app_id")
  private String appId;

  @NotNull
  @Column(name = "policy_json")
  private String policyJson;

  @NotNull
  @Column(name = "guid")
  private String guid;

  public String getAppId() {
    return appId;
  }

  public void setAppId(String appId) {
    this.appId = appId;
  }

  public String getPolicyJson() {
    return policyJson;
  }

  public void setPolicyJson(String policyJson) {
    this.policyJson = policyJson;
  }

  public String getGuid() {
    return guid;
  }

  public void setGuid(String guid) {
    this.guid = guid;
  }

  @Override
  public PolicyJsonEntity mapRow(ResultSet rs, int rowNum) throws SQLException {
    PolicyJsonEntity policyJsonEntity = new PolicyJsonEntity();
    policyJsonEntity.setAppId(rs.getString("app_id"));
    policyJsonEntity.setPolicyJson(rs.getString("policy_json"));
    policyJsonEntity.setGuid(rs.getString("guid"));
    return policyJsonEntity;
  }
}
