package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.List;

import javax.annotation.Resource;
import javax.sql.DataSource;

import org.cloudfoundry.autoscaler.scheduler.entity.PolicyJsonEntity;
import org.springframework.jdbc.core.support.JdbcDaoSupport;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

@Repository("policyJsonDao")
public class PolicyJsonDaoImpl extends JdbcDaoSupport implements PolicyJsonDao {

	@Resource(name = "policyDbDataSource")
	private void setupPolicyDataSource(DataSource policyDbDataSource) {
		setDataSource(policyDbDataSource);
	}

	@Override
	@Transactional(readOnly = true)
	public List<PolicyJsonEntity> getAllPolicies() {
		List<PolicyJsonEntity> list = getJdbcTemplate().query("SELECT * FROM policy_json", new PolicyJsonEntity());
		return list;
	}

}
