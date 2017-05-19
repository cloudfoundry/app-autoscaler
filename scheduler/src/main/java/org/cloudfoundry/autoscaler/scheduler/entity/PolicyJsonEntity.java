package org.cloudfoundry.autoscaler.scheduler.entity;

import java.io.IOException;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.List;

import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.Id;
import javax.persistence.Table;
import javax.persistence.Transient;
import javax.validation.constraints.NotNull;

import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;
import org.springframework.jdbc.core.RowMapper;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;

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

	@Transient
	private ApplicationSchedules schedules;

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

	public ApplicationSchedules getSchedules() {
		return schedules;
	}

	public void setSchedules(ApplicationSchedules schedules) {
		this.schedules = schedules;
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
		ObjectMapper mapper = new ObjectMapper();
		//do not mapper scaling_rules in policy_json
		mapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
		try {
			policyJsonEntity.setSchedules(mapper.readValue(policyJsonEntity.getPolicyJson(), ApplicationSchedules.class));
			if(policyJsonEntity.getSchedules()!=null && policyJsonEntity.getSchedules().getSchedules()!=null&&policyJsonEntity.getSchedules().getSchedules().hasSchedules()){
				Schedules schedules = policyJsonEntity.getSchedules().getSchedules();
				List<RecurringScheduleEntity> recurringSchedules = schedules.getRecurringSchedule();
				List<SpecificDateScheduleEntity> specificDateSchedules = schedules.getSpecificDate();
				if(recurringSchedules != null){
					for(RecurringScheduleEntity recurring : recurringSchedules){
						recurring.setAppId(policyJsonEntity.getAppId());
						recurring.setTimeZone(schedules.getTimeZone());
						recurring.setDefaultInstanceMinCount(policyJsonEntity.getSchedules().getInstanceMinCount());
						recurring.setDefaultInstanceMaxCount(policyJsonEntity.getSchedules().getInstanceMaxCount());
						recurring.setGuid(policyJsonEntity.getGuid());
					}
				}
				if(specificDateSchedules != null){
					for(SpecificDateScheduleEntity specificDate : specificDateSchedules){
						specificDate.setAppId(policyJsonEntity.getAppId());
						specificDate.setTimeZone(schedules.getTimeZone());
						specificDate.setDefaultInstanceMinCount(policyJsonEntity.getSchedules().getInstanceMinCount());
						specificDate.setDefaultInstanceMaxCount(policyJsonEntity.getSchedules().getInstanceMaxCount());
						specificDate.setGuid(policyJsonEntity.getGuid());
					}
				}
				 
			}
			
		} catch (IOException e) {
			e.printStackTrace();
			policyJsonEntity.setSchedules(null);
		}
		return policyJsonEntity;
	}

}
