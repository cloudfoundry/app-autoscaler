package org.cloudfoundry.autoscaler.scheduler.dao;

import jakarta.annotation.Resource;
import java.util.List;
import javax.sql.DataSource;
import org.cloudfoundry.autoscaler.scheduler.entity.PolicyJsonEntity;
import org.springframework.jdbc.core.support.JdbcDaoSupport;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

@Repository("policyJsonDao")
public class PolicyJsonDaoImpl extends JdbcDaoSupport implements PolicyJsonDao {

  private static final String SQL_GET_All_POLICY_JSON = "SELECT * FROM policy_json";

  @Resource(name = "policyDbDataSource")
  private void setupPolicyDataSource(DataSource policyDbDataSource) {
    setDataSource(policyDbDataSource);
  }

  @Override
  @Transactional(readOnly = true)
  public List<PolicyJsonEntity> getAllPolicies() {
    return getJdbcTemplate().query(SQL_GET_All_POLICY_JSON, new PolicyJsonEntity());
  }
}
