package org.cloudfoundry.autoscaler;

import java.util.List;
import java.util.Map;

import org.junit.Assert;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * This test class checks the consistency of the db.changelog-master.yaml file.
 * 
 * @author pradyutsarma
 */
@RunWith(SpringRunner.class)
@TestPropertySource(locations = "classpath:test.properties")
@SpringBootTest
public class DbApplicationTests {

	@Autowired
	private JdbcTemplate jdbcTemplate;

	@Test
	public void testPolicyDB() {

		// Insert some data to the policyDB
		String insertIntoDBQuery = "insert into policy_json ( app_id, policy_json ) values ('myApp', '{id:policyId}')";
		jdbcTemplate.execute(insertIntoDBQuery);

		// Fetch the data from the policyDB
		String selectFromDBQuery = "select * from public.policy_json";
		List<Map<String, Object>> resultSet = jdbcTemplate.queryForList(selectFromDBQuery);

		// Assert the retrieval
		Assert.assertTrue(resultSet.size() == 1);

	}

}
