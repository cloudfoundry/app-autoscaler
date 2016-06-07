package org.cloudfoundry.autoscaler.scheduler.service;

import java.sql.Time;
import java.util.ArrayList;
import java.util.List;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.cloudfoundry.autoscaler.scheduler.dao.ScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.SpecificDateSchedule;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.JobScheduleTypeEnum;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * Service class to persist the schedule entity in the database and create
 * scheduled job.
 * 
 * @author Fujitsu
 *
 */
@Service
public class ScalingScheduleManager {

	@Autowired
	private ScheduleDao scheduleDao;
	@Autowired
	private ScalingJobManager scalingJobManager;
	private Log logger = LogFactory.getLog(this.getClass());

	/**
	 * TODO: Created just a place holder, method will require to call dao and
	 * fetch all the schedules for the specified application id.
	 * 
	 * @param appId
	 * @return
	 * @throws Exception
	 */
	public ApplicationScalingSchedules getSchedules(String appId) {
		List<ScheduleEntity> allScheduleEntitiesForApp = scheduleDao.findAllSchedulesByAppId(appId);
		ApplicationScalingSchedules applicationScalinSchedules = populateScheduleModel(appId,
				allScheduleEntitiesForApp);
		return applicationScalinSchedules;
	}

	/**
	 * Calls private helper methods to persist the schedules in the database and
	 * calls ScalingJobManager to create scaling action jobs.
	 * 
	 * @param applicationScalinSchedules
	 * @throws Exception
	 */
	@Transactional
	public String createSchedules(ApplicationScalingSchedules applicationScalinSchedules) {

		String appId = applicationScalinSchedules.getApp_id();
		String timeZone = applicationScalinSchedules.getTimezone();
		List<SpecificDateSchedule> specificDateSchedules = applicationScalinSchedules.getSpecific_date();
		try {
			for (SpecificDateSchedule specificDateSchedule : specificDateSchedules) {

				// Set up the schedule entity from model object
				ScheduleEntity scheduleEntity = populateScheduleEntity(appId, timeZone, specificDateSchedule,
						JobScheduleTypeEnum.SIMPLE);

				// Persist the schedule in database
				saveNewSpecificDateSchedule(scheduleEntity);

				// Ask ScalingJobManager to create scaling job
				scalingJobManager.createSimpleJob(scheduleEntity);
			}
		} catch (Exception x) {
			logger.error(x.getMessage(), x);
			return null;
		}
		return appId;
	}

	/**
	 * Persist the application scaling schedule entity.
	 * 
	 * @param scheduleEntity
	 * @return
	 */
	private ScheduleEntity saveNewSpecificDateSchedule(ScheduleEntity scheduleEntity) {
		ScheduleEntity savedScheduleEntity = scheduleDao.create(scheduleEntity);
		return savedScheduleEntity;
	}

	/**
	 * Helper method to extract the data from the specific date schedule model
	 * object, populates the schedule entity.
	 * 
	 * @param appId
	 * @param timeZone
	 * @param specificDateSchedule
	 * @param jobSchedulType
	 * @return
	 */
	private ScheduleEntity populateScheduleEntity(String appId, String timeZone,
			SpecificDateSchedule specificDateSchedule, JobScheduleTypeEnum jobSchedulType) {

		ScheduleEntity scheduleEntity = new ScheduleEntity();
		scheduleEntity.setAppId(appId);
		scheduleEntity.setTimezone(timeZone);
		scheduleEntity.setStartDate(java.sql.Date.valueOf(specificDateSchedule.getStart_date()));
		scheduleEntity.setStartTime(Time.valueOf(specificDateSchedule.getStart_time()));
		scheduleEntity.setEndDate(java.sql.Date.valueOf(specificDateSchedule.getEnd_date()));
		scheduleEntity.setEndTime(Time.valueOf(specificDateSchedule.getEnd_time()));
		scheduleEntity.setInstanceMaxCount(specificDateSchedule.getInstance_max_count());
		scheduleEntity.setInstanceMinCount(specificDateSchedule.getInstance_min_count());
		scheduleEntity.setJobScheduleType(jobSchedulType.getDbValue());

		return scheduleEntity;

	}

	/**
	 * Helper method to extract the data from the schedule entity collection,
	 * and populate schedule model.
	 * 
	 * @param scheduleEntity
	 * @return
	 */
	private ApplicationScalingSchedules populateScheduleModel(String appId,
			List<ScheduleEntity> allScheduleEntitiesForApp) {

		ApplicationScalingSchedules applicationScalingSchedules = new ApplicationScalingSchedules();
		applicationScalingSchedules.setApp_id(appId);
		if (!allScheduleEntitiesForApp.isEmpty()) {
			applicationScalingSchedules.setTimezone(allScheduleEntitiesForApp.get(0).getTimezone());
		}

		List<SpecificDateSchedule> specificDateSchedules = new ArrayList<SpecificDateSchedule>();
		applicationScalingSchedules.setSpecific_date(specificDateSchedules);
		for (ScheduleEntity scheduleEntity : allScheduleEntitiesForApp) {
			switch (JobScheduleTypeEnum.getEnum(scheduleEntity.getJobScheduleType())) {
			case SIMPLE:
				SpecificDateSchedule specificDateSchedule = new SpecificDateSchedule();
				specificDateSchedule.setStart_date(DateHelper.convertDateToString(scheduleEntity.getStartDate()));
				specificDateSchedule.setEnd_date(DateHelper.convertDateToString(scheduleEntity.getEndDate()));
				specificDateSchedule.setStart_time(DateHelper.convertTimeToString(scheduleEntity.getStartTime()));
				specificDateSchedule.setEnd_time(DateHelper.convertTimeToString(scheduleEntity.getEndTime()));

				specificDateSchedule.setInstance_min_count(scheduleEntity.getInstanceMinCount());
				specificDateSchedule.setInstance_max_count(scheduleEntity.getInstanceMaxCount());

				specificDateSchedules.add(specificDateSchedule);
				break;
			default:
				break;
			}
		}

		return applicationScalingSchedules;

	}
}
