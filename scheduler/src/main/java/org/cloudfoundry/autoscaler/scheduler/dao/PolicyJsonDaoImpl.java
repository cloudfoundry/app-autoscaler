package org.cloudfoundry.autoscaler.scheduler.dao;

import jakarta.annotation.Resource;
import java.util.List;
import javax.sql.DataSource;
import org.cloudfoundry.autoscaler.scheduler.entity.PolicyJsonEntity;
import org.springframework.boot.sql.init.dependency.DependsOnDatabaseInitialization;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

@DependsOnDatabaseInitialization
@Repository("policyJsonDao")
public class PolicyJsonDaoImpl implements PolicyJsonDao {

  private static final String SQL_GET_All_POLICY_JSON = "SELECT * FROM policy_json";

  private JdbcTemplate jdbcTemplate;

  @Resource(name = "policyDbDataSource")
  private void setupPolicyDataSource(DataSource policyDbDataSource) {
    jdbcTemplate = new JdbcTemplate(policyDbDataSource);
  }

  @Override
  @Transactional(readOnly = true)
  public List<PolicyJsonEntity> getAllPolicies() {
    return jdbcTemplate.query(SQL_GET_All_POLICY_JSON, new PolicyJsonEntity());
  }
}
