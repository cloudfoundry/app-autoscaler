package org.cloudfoundry.autoscaler.scheduler.service;

import java.io.IOException;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.TimeZone;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.PolicyJsonDao;
import org.cloudfoundry.autoscaler.scheduler.dao.RecurringScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.PolicyJsonEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.SynchronizeResult;
import org.cloudfoundry.autoscaler.scheduler.util.DataValidationHelper;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.RecurringScheduleTime;
import org.cloudfoundry.autoscaler.scheduler.util.ScalingEngineUtil;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.SpecificDateScheduleDateTime;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.SchedulerInternalException;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.client.HttpStatusCodeException;
import org.springframework.web.client.ResourceAccessException;
import org.springframework.web.client.RestOperations;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;

/**
 * Service class to persist the schedule entity in the database and create
 * scheduled job.
 */
@Service
public class ScheduleManager {

	@Autowired
	private SpecificDateScheduleDao specificDateScheduleDao;
	@Autowired
	private RecurringScheduleDao recurringScheduleDao;
	@Autowired
	private ActiveScheduleDao activeScheduleDao;
	@Autowired
	private PolicyJsonDao policyJsonDao;
	@Autowired
	private ScheduleJobManager scheduleJobManager;
	@Autowired
	private RestOperations restOperations;
	@Autowired
	private ValidationErrorResult validationErrorResult;
	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Value("${autoscaler.scalingengine.url}")
	private String scalingEngineUrl;

	private Logger logger = LogManager.getLogger(this.getClass());

	/**
	 * Calls dao and fetch all the schedules for the specified application id.
	 *
	 * @param appId
	 * @return
	 */
	public ApplicationSchedules getAllSchedules(String appId) {
		logger.info("Get All schedules for application: " + appId);

		ApplicationSchedules applicationSchedules = new ApplicationSchedules();
		Schedules schedules = new Schedules();
		applicationSchedules.setSchedules(schedules);
		List<SpecificDateScheduleEntity> allSpecificDateScheduleEntitiesForApp;
		List<RecurringScheduleEntity> allRecurringScheduleEntitiesForApp;

		try {
			allSpecificDateScheduleEntitiesForApp = specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId);
			if (!allSpecificDateScheduleEntitiesForApp.isEmpty()) {
				schedules.setSpecificDate(allSpecificDateScheduleEntitiesForApp);
			}

			allRecurringScheduleEntitiesForApp = recurringScheduleDao.findAllRecurringSchedulesByAppId(appId);
			if (!allRecurringScheduleEntitiesForApp.isEmpty()) {
				schedules.setRecurringSchedule(allRecurringScheduleEntitiesForApp);
			}

		} catch (DatabaseValidationException dve) {

			validationErrorResult.addErrorForDatabaseValidationException(dve, "database.error.get.failed",
					"app_id=" + appId);
			throw new SchedulerInternalException("Database error", dve);
		}

		return applicationSchedules;
	}

	/**
	 * This method calls the helper method to sets up the basic common information in the schedule entities.
	 * @param appId
	 * @param applicationPolicy
	 */
	public void setUpSchedules(String appId, String guid, ApplicationSchedules applicationPolicy) {

		// If there are schedules then only set the meta data in the schedule entities
		if (applicationPolicy.getSchedules() != null && applicationPolicy.getSchedules().hasSchedules()) {
			List<SpecificDateScheduleEntity> specificDateSchedules = applicationPolicy.getSchedules().getSpecificDate();
			if (specificDateSchedules != null) {
				for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateSchedules) {
					// Sets the meta data in specific date schedules list
					setUpSchedules(appId, guid, applicationPolicy, specificDateScheduleEntity);
				}
			}

			// Call the setUpSchedules to set the meta data in recurring schedules list
			List<RecurringScheduleEntity> recurringSchedules = applicationPolicy.getSchedules().getRecurringSchedule();
			if (recurringSchedules != null) {
				for (RecurringScheduleEntity recurringScheduleEntity : recurringSchedules) {
					setUpSchedules(appId, guid, applicationPolicy, recurringScheduleEntity);
				}
			}
		}
	}

	/**
	 * Sets the meta data(like the appId, timeZone etc) in the specified entity.
	 * @param appId
	 * @param applicationPolicy
	 * @param scheduleEntity
	 */
	private void setUpSchedules(String appId, String guid, ApplicationSchedules applicationPolicy, ScheduleEntity scheduleEntity) {
		scheduleEntity.setAppId(appId);
		scheduleEntity.setTimeZone(applicationPolicy.getSchedules().getTimeZone());
		scheduleEntity.setDefaultInstanceMinCount(applicationPolicy.getInstanceMinCount());
		scheduleEntity.setDefaultInstanceMaxCount(applicationPolicy.getInstanceMaxCount());
		scheduleEntity.setGuid(guid);

	}

	/**
	 * This method does the basic data validation and calls the helper method to
	 * do further validation.
	 *
	 * @param appId
	 * @param applicationPolicy
	 */
	public void validateSchedules(String appId, ApplicationSchedules applicationPolicy) {
		logger.info("Validate schedules for application: " + appId);

		// Validate the application id
		if (!DataValidationHelper.isNotEmpty(appId)) {
			validationErrorResult.addFieldError(applicationPolicy, "data.value.not.specified", "app_id");
		}

		// Validate the default minimum and maximum instance count
		validateDefaultInstanceMinMaxCount(applicationPolicy.getInstanceMinCount(),
				applicationPolicy.getInstanceMaxCount());

		// Validate schedules.
		Schedules schedules = applicationPolicy.getSchedules();
		if (schedules != null ) {
			boolean isValidTimeZone = validateTimeZone(applicationPolicy);

			if (schedules.hasSchedules()) {
				List<SpecificDateScheduleEntity> specificDateSchedules = applicationPolicy.getSchedules().getSpecificDate();
				// Validate specific date schedules.
				if (specificDateSchedules != null) {
					validateSpecificDateSchedules(specificDateSchedules, isValidTimeZone);
				}

				List<RecurringScheduleEntity> recurringSchedules = applicationPolicy.getSchedules().getRecurringSchedule();
				// Validate recurring schedules.
				if (recurringSchedules != null) {
					validateRecurringSchedules(recurringSchedules, isValidTimeZone);
				}
			}
		} 

	}

	private boolean validateTimeZone(ApplicationSchedules applicationPolicy) {
		String timeZoneId = applicationPolicy.getSchedules().getTimeZone();

		boolean isValidTimeZone = true;
		if (!DataValidationHelper.isNotEmpty(timeZoneId)) {
			validationErrorResult.addFieldError(applicationPolicy, "data.value.not.specified.timezone", "timeZone");
			isValidTimeZone = false;
		}

		if (isValidTimeZone && !DataValidationHelper.isValidTimeZone(timeZoneId)) {
			validationErrorResult.addFieldError(applicationPolicy, "data.invalid.timezone", "timeZone", timeZoneId);
			isValidTimeZone = false;
		}
		return isValidTimeZone;
	}

	/**
	 * This method traverses through the list and calls helper methods to perform validations on
	 * the specific date schedule entity.
	 *
	 * @param specificDateSchedules
	 * @param isValidTimeZone
	 */
	private void validateSpecificDateSchedules(List<SpecificDateScheduleEntity> specificDateSchedules,
			boolean isValidTimeZone) {
		List<SpecificDateScheduleDateTime> scheduleStartEndTimeList = new ArrayList<>();

		// Identifier to tell which schedule is being validated, will be used in the validation messages
		// convenience to identify the schedule that has an issue. First schedule identified as 0
		int scheduleIdentifier = 0;
		for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateSchedules) {

			String scheduleBeingProcessed = ScheduleTypeEnum.SPECIFIC_DATE.getDescription() + " " + scheduleIdentifier; // Specific date/Recurring and the index of schedule being processed

			// Validate the dates and times only if the time zone is valid
			if (isValidTimeZone) {
				// Call helper method to validate the start date time and end date time.
				SpecificDateScheduleDateTime validScheduleDateTime = validateStartEndDateTime(scheduleBeingProcessed,
						specificDateScheduleEntity);

				if (validScheduleDateTime != null) {
					scheduleStartEndTimeList.add(validScheduleDateTime);

				}
			}

			Integer initialMinInstanceCount = specificDateScheduleEntity.getInitialMinInstanceCount();
			// The initial minimum instance count cannot be negative.
			if (DataValidationHelper.isNotNull(initialMinInstanceCount) && initialMinInstanceCount < 0) {
				validationErrorResult.addFieldError(null, "schedule.data.value.invalid", scheduleBeingProcessed,
						"initial_min_instance_count", initialMinInstanceCount);
			}

			// Validate instance minimum count and maximum count.
			validateInstanceMinMaxCount(scheduleBeingProcessed, specificDateScheduleEntity.getInstanceMinCount(),
					specificDateScheduleEntity.getInstanceMaxCount());
			++scheduleIdentifier;
		}

		// Validate the dates for overlap
		if (!scheduleStartEndTimeList.isEmpty()) {
			List<String[]> overlapDateTimeValidationErrorMsgList = DataValidationHelper
					.isNotOverlapForSpecificDate(scheduleStartEndTimeList);
			for (String[] arguments : overlapDateTimeValidationErrorMsgList) {
				validationErrorResult.addFieldError(specificDateSchedules, "schedule.date.overlap",
						(Object[]) arguments);
			}
		}

	}

	/**
	 * This method traverses through the list and calls helper methods to perform validations on
	 * the recurring schedule entity.
	 *
	 * @param recurringSchedules
	 * @param isValidTimeZone
	 */
	private void validateRecurringSchedules(List<RecurringScheduleEntity> recurringSchedules, boolean isValidTimeZone) {
		int scheduleIdentifier = 0;

		List<RecurringScheduleTime> recurringScheduleTimes = new ArrayList<>();

		for (RecurringScheduleEntity recurringScheduleEntity : recurringSchedules) {
			String scheduleBeingProcessed = ScheduleTypeEnum.RECURRING.getDescription() + " " + scheduleIdentifier; // Recurring

			if (isValidTimeZone) {
				RecurringScheduleTime scheduleTime = validateRecurringScheduleTime(scheduleBeingProcessed,
						recurringScheduleEntity);
				if (scheduleTime != null) {
					recurringScheduleTimes.add(scheduleTime);
				}
			}

			Integer initialMinInstanceCount = recurringScheduleEntity.getInitialMinInstanceCount();
			// The initial minimum instance count cannot be negative.
			if (DataValidationHelper.isNotNull(initialMinInstanceCount) && initialMinInstanceCount < 0) {
				validationErrorResult.addFieldError(null, "schedule.data.value.invalid", scheduleBeingProcessed,
						"initial_min_instance_count", initialMinInstanceCount);
			}

			// Validate instance minimum count and maximum count.
			validateInstanceMinMaxCount(scheduleBeingProcessed, recurringScheduleEntity.getInstanceMinCount(),
					recurringScheduleEntity.getInstanceMaxCount());
			++scheduleIdentifier;
		}
		if (isValidTimeZone) {
			// Call helper method to validate the start date time and end date time.
			List<String[]> messages = DataValidationHelper.isNotOverlapRecurringSchedules(recurringScheduleTimes);
			for (String[] arguments : messages) {
				validationErrorResult.addFieldError(recurringScheduleTimes, "schedule.date.overlap",
						(Object[]) arguments);
			}
		}
	}

	private RecurringScheduleTime validateRecurringScheduleTime(String scheduleBeingProcessed,
			RecurringScheduleEntity recurringSchedule) {
		boolean isValid = true;

		if (!validateDayOfWeekOrMonth(scheduleBeingProcessed, recurringSchedule)) {
			isValid = false;
		}

		if (!validateStartEndDate(scheduleBeingProcessed, recurringSchedule)) {
			isValid = false;
		}

		if (!validateStartEndTime(scheduleBeingProcessed, recurringSchedule)) {
			isValid = false;
		}

		RecurringScheduleTime time = null;
		if (isValid) {
			time = new RecurringScheduleTime(scheduleBeingProcessed, recurringSchedule);
		}

		return time;
	}

	private boolean validateStartEndTime(String scheduleBeingProcessed, RecurringScheduleEntity recurringSchedule) {
		boolean isValid = true;
		LocalTime startTime = recurringSchedule.getStartTime();
		LocalTime endTime = recurringSchedule.getEndTime();

		if (!DataValidationHelper.isNotNull(startTime)) {
			isValid = false;
			validationErrorResult.addFieldError(recurringSchedule, "schedule.data.value.not.specified",
					scheduleBeingProcessed, "start_time");
		}

		if (!DataValidationHelper.isNotNull(endTime)) {
			isValid = false;
			validationErrorResult.addFieldError(recurringSchedule, "schedule.data.value.not.specified",
					scheduleBeingProcessed, "end_time");
		}

		if (isValid) {
			// If end date time is not after start date time, then dates invalid
			if (!DataValidationHelper.isAfter(endTime, startTime)) {
				isValid = false;
				validationErrorResult.addFieldError(recurringSchedule, "schedule.date.invalid.start.after.end",
						scheduleBeingProcessed, "end_time", DateHelper.convertLocalTimeToString(endTime), "start_time",
						DateHelper.convertLocalTimeToString(startTime));
			}
		}
		return isValid;
	}

	private boolean validateStartEndDate(String scheduleBeingProcessed, RecurringScheduleEntity recurringSchedule) {
		// Note: For recurring schedule, start and end date are optional so not checking for null
		boolean isValid = true;
		LocalDate startDate = recurringSchedule.getStartDate();
		LocalDate endDate = recurringSchedule.getEndDate();
		TimeZone timeZone = TimeZone.getTimeZone(recurringSchedule.getTimeZone());

		if (startDate != null) {
			// it should be after current date.
			if (!DataValidationHelper.isDateAfterOrEqualsNow(startDate, timeZone)) {
				isValid = false;
				validationErrorResult.addFieldError(recurringSchedule, "schedule.date.invalid.before.current",
						scheduleBeingProcessed, "start_date", startDate);
			}
		}

		if (endDate != null) {
			// it should be after current date.
			if (!DataValidationHelper.isDateAfterOrEqualsNow(endDate, timeZone)) {
				isValid = false;
				validationErrorResult.addFieldError(recurringSchedule, "schedule.date.invalid.before.current",
						scheduleBeingProcessed, "end_date", endDate);
			}
		}

		if (startDate != null && endDate != null && isValid) {
			// startDate should be before or equal to endDate
			if (startDate.compareTo(endDate) > 0) {
				isValid = false;
				validationErrorResult.addFieldError(recurringSchedule, "schedule.date.invalid.end.before.start",
						scheduleBeingProcessed, "end_date", endDate, "start_date", startDate);
			}
		}
		return isValid;
	}

	private boolean validateDayOfWeekOrMonth(String scheduleBeingProcessed, RecurringScheduleEntity recurringSchedule) {
		boolean isValid = true;
		int[] dayOfMonth = recurringSchedule.getDaysOfMonth();
		int[] dayOfWeek = recurringSchedule.getDaysOfWeek();

		if (!DataValidationHelper.isNotEmpty(dayOfMonth) && !DataValidationHelper.isNotEmpty(dayOfWeek)) {
			isValid = false;
			validationErrorResult.addFieldError(recurringSchedule, "schedule.data.both.values.not.specified",
					scheduleBeingProcessed, "day_of_week", "day_of_month");
		}

		if (DataValidationHelper.isNotEmpty(dayOfMonth) && DataValidationHelper.isNotEmpty(dayOfWeek)) {
			isValid = false;
			validationErrorResult.addFieldError(recurringSchedule, "schedule.data.both.values.specified",
					scheduleBeingProcessed, "day_of_week", "day_of_month");
		}

		if (DataValidationHelper.isNotEmpty(dayOfWeek)) {
			if (!DataValidationHelper.isBetweenMinAndMaxValues(dayOfWeek, DateHelper.DAY_OF_WEEK_MINIMUM,
					DateHelper.DAY_OF_WEEK_MAXIMUM)) {
				isValid = false;
				validationErrorResult.addFieldError(recurringSchedule, "schedule.data.invalid.day",
						scheduleBeingProcessed, "day_of_week", DateHelper.DAY_OF_WEEK_MINIMUM,
						DateHelper.DAY_OF_WEEK_MAXIMUM);
			}

			if (!DataValidationHelper.isElementUnique(dayOfWeek)) {
				isValid = false;
				validationErrorResult.addFieldError(recurringSchedule, "schedule.data.not.unique",
						scheduleBeingProcessed, "day_of_week");
			}
		}

		if (DataValidationHelper.isNotEmpty(dayOfMonth)) {
			if (!DataValidationHelper.isBetweenMinAndMaxValues(dayOfMonth, DateHelper.DAY_OF_MONTH_MINIMUM,
					DateHelper.DAY_OF_MONTH_MAXIMUM)) {
				isValid = false;
				validationErrorResult.addFieldError(recurringSchedule, "schedule.data.invalid.day",
						scheduleBeingProcessed, "day_of_month", DateHelper.DAY_OF_MONTH_MINIMUM,
						DateHelper.DAY_OF_MONTH_MAXIMUM);
			}

			if (!DataValidationHelper.isElementUnique(dayOfMonth)) {
				isValid = false;
				validationErrorResult.addFieldError(recurringSchedule, "schedule.data.not.unique",
						scheduleBeingProcessed, "day_of_month");
			}
		}
		return isValid;
	}

	/**
	 * This method validates the default instance minimum and maximum count.
	 *
	 * @param defaultInstanceMinCount
	 * @param defaultInstanceMaxCount
	 */
	private void validateDefaultInstanceMinMaxCount(Integer defaultInstanceMinCount, Integer defaultInstanceMaxCount) {

		boolean isValid = true;

		boolean isValidInstanceCount = DataValidationHelper.isNotNull(defaultInstanceMinCount);
		// The minimum instance count cannot be null.
		if (!isValidInstanceCount) {
			validationErrorResult.addFieldError(null, "data.default.value.not.specified", "instance_min_count");
			isValid = false;
		}

		// The minimum instance count cannot be negative.
		if (isValidInstanceCount && defaultInstanceMinCount < 0) {
			validationErrorResult.addFieldError(null, "data.default.value.invalid", "instance_min_count",
					defaultInstanceMinCount);
			isValid = false;
		}

		isValidInstanceCount = DataValidationHelper.isNotNull(defaultInstanceMaxCount);
		// The maximum instance count cannot be null.
		if (!isValidInstanceCount) {
			validationErrorResult.addFieldError(null, "data.default.value.not.specified", "instance_max_count");
			isValid = false;
		}

		// The maximum instance count cannot be zero or negative.
		if (isValidInstanceCount && defaultInstanceMaxCount <= 0) {
			validationErrorResult.addFieldError(null, "data.default.value.invalid", "instance_max_count",
					defaultInstanceMaxCount);
			isValid = false;
		}

		if (isValid) {
			// Check the maximum instance count is greater than minimum instance count
			if (defaultInstanceMaxCount <= defaultInstanceMinCount) {
				validationErrorResult.addFieldError(null, "data.default.instanceCount.invalid.min.greater",
						"instance_max_count", defaultInstanceMaxCount, "instance_min_count", defaultInstanceMinCount);
			}
		}
	}

	/**
	 * This method validates the instance minimum and maximum count.
	 *
	 * @param scheduleBeingProcessed
	 * @param instanceMinCount
	 * @param instanceMaxCount
	 */
	private void validateInstanceMinMaxCount(String scheduleBeingProcessed, Integer instanceMinCount,
			Integer instanceMaxCount) {

		boolean isValid = true;

		boolean isValidInstanceCount = DataValidationHelper.isNotNull(instanceMinCount);
		// The minimum instance count cannot be null.
		if (!isValidInstanceCount) {
			validationErrorResult.addFieldError(null, "schedule.data.value.not.specified", scheduleBeingProcessed,
					"instance_min_count");
			isValid = false;
		}

		// The minimum instance count cannot be negative.
		if (isValidInstanceCount && instanceMinCount < 0) {
			validationErrorResult.addFieldError(null, "schedule.data.value.invalid", scheduleBeingProcessed,
					"instance_min_count", instanceMinCount);
			isValid = false;
		}

		isValidInstanceCount = DataValidationHelper.isNotNull(instanceMaxCount);
		// The maximum instance count cannot be null.
		if (!isValidInstanceCount) {
			validationErrorResult.addFieldError(null, "schedule.data.value.not.specified", scheduleBeingProcessed,
					"instance_max_count");
			isValid = false;
		}

		// The maximum instance count cannot be zero or negative.
		if (isValidInstanceCount && instanceMaxCount <= 0) {
			validationErrorResult.addFieldError(null, "schedule.data.value.invalid", scheduleBeingProcessed,
					"instance_max_count", instanceMaxCount);
			isValid = false;
		}

		if (isValid) {
			// Check the maximum instance count is greater than minimum instance count
			if (instanceMaxCount <= instanceMinCount) {
				validationErrorResult.addFieldError(null, "schedule.instanceCount.invalid.min.greater",
						scheduleBeingProcessed, "instance_max_count", instanceMaxCount, "instance_min_count",
						instanceMinCount);
			}
		}
	}

	/**
	 * This method validates the start date time and end date time of the
	 * specified specific schedule.
	 *
	 * @param specificDateSchedule
	 * @return
	 */
	private SpecificDateScheduleDateTime validateStartEndDateTime(String scheduleBeingProcessed,
			SpecificDateScheduleEntity specificDateSchedule) {
		boolean isValid = true;
		SpecificDateScheduleDateTime validScheduleDateTime = null;

		LocalDateTime startDateTime = specificDateSchedule.getStartDateTime();
		LocalDateTime endDateTime = specificDateSchedule.getEndDateTime();

		TimeZone timeZone = TimeZone.getTimeZone(specificDateSchedule.getTimeZone());

		boolean isValidDtTm = DataValidationHelper.isNotNull(startDateTime);
		if (!isValidDtTm) {
			isValid = false;
			validationErrorResult.addFieldError(specificDateSchedule, "schedule.data.value.not.specified",
					scheduleBeingProcessed, "start_date_time");
		}

		if (isValidDtTm) {
			// Check the start date time is after current date time
			if (!DataValidationHelper.isDateTimeAfterNow(startDateTime, timeZone)) {
				isValid = false;
				validationErrorResult.addFieldError(specificDateSchedule, "schedule.date.invalid.current.after",
						scheduleBeingProcessed, "start_date_time",
						DateHelper.convertLocalDateTimeToString(startDateTime));
			}
		}

		isValidDtTm = DataValidationHelper.isNotNull(endDateTime);
		if (!isValidDtTm) {
			isValid = false;
			validationErrorResult.addFieldError(specificDateSchedule, "schedule.data.value.not.specified",
					scheduleBeingProcessed, "end_date_time");
		}

		if (isValidDtTm) {
			// Check the end date time is after current date time
			if (!DataValidationHelper.isDateTimeAfterNow(endDateTime, timeZone)) {
				isValid = false;
				validationErrorResult.addFieldError(specificDateSchedule, "schedule.date.invalid.current.after",
						scheduleBeingProcessed, "end_date_time", DateHelper.convertLocalDateTimeToString(endDateTime));
			}

		}

		// Check the end date is after the start date
		if (isValid) {

			// If end date time is not after start date time, then dates invalid
			if (!DataValidationHelper.isAfter(endDateTime, startDateTime)) {
				validationErrorResult.addFieldError(specificDateSchedule, "schedule.date.invalid.start.after.end",
						scheduleBeingProcessed, "end_date_time", DateHelper.convertLocalDateTimeToString(endDateTime),
						"start_date_time", DateHelper.convertLocalDateTimeToString(startDateTime));
			} else {
				validScheduleDateTime = new SpecificDateScheduleDateTime(scheduleBeingProcessed, startDateTime,
						endDateTime);
			}
		}

		return validScheduleDateTime;
	}

	/**
	 * Calls private helper methods to persist the schedules in the database and
	 * calls ScalingJobManager to create scaling action jobs.
	 *
	 * @param schedules
	 */
	@Transactional
	public void createSchedules(Schedules schedules) {

		List<SpecificDateScheduleEntity> specificDateSchedules = schedules.getSpecificDate();
		if (specificDateSchedules != null) {
			for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateSchedules) {
				// Persist the schedule in database
				SpecificDateScheduleEntity savedScheduleEntity = saveNewSpecificDateSchedule(
						specificDateScheduleEntity);

				// Ask ScalingJobManager to create scaling job
				if (savedScheduleEntity != null) {
					scheduleJobManager.createSimpleJob(savedScheduleEntity);
				}
			}
		}

		List<RecurringScheduleEntity> recurringSchedules = schedules.getRecurringSchedule();
		if (recurringSchedules != null) {
			for (RecurringScheduleEntity recurringScheduleEntity : recurringSchedules) {
				// Persist the schedule in database
				RecurringScheduleEntity savedScheduleEntity = saveNewRecurringSchedule(recurringScheduleEntity);

				// Ask ScalingJobManager to create scaling job
				if (savedScheduleEntity != null) {
					scheduleJobManager.createCronJob(savedScheduleEntity);
				}
			}
		}
	}

	/**
	 * Persist the schedule entity holding the application's specific date scheduling information.
	 *
	 * @param specificDateScheduleEntity
	 * @return
	 */
	private SpecificDateScheduleEntity saveNewSpecificDateSchedule(
			SpecificDateScheduleEntity specificDateScheduleEntity) {
		SpecificDateScheduleEntity savedScheduleEntity;
		try {
			savedScheduleEntity = specificDateScheduleDao.create(specificDateScheduleEntity);

		} catch (DatabaseValidationException dve) {

			validationErrorResult.addErrorForDatabaseValidationException(dve, "database.error.create.failed",
					"app_id=" + specificDateScheduleEntity.getAppId());
			throw new SchedulerInternalException("Database error", dve);
		}
		return savedScheduleEntity;
	}

	private RecurringScheduleEntity saveNewRecurringSchedule(RecurringScheduleEntity recurringScheduleEntity) {
		RecurringScheduleEntity savedScheduleEntity;
		try {
			savedScheduleEntity = recurringScheduleDao.create(recurringScheduleEntity);
		} catch (DatabaseValidationException dve) {
			validationErrorResult.addErrorForDatabaseValidationException(dve, "database.error.create.failed",
					"app_id=" + recurringScheduleEntity.getAppId());
			throw new SchedulerInternalException("Database error", dve);
		}
		return savedScheduleEntity;
	}

	/**
	 * Calls private helper methods to delete the schedules from the database and
	 * calls ScalingJobManager to delete scaling action jobs.
	 *
	 * @param appId
	 */
	@Transactional
	public void deleteSchedules(String appId) {

		// Get all the specific date schedules for the specifies application id and delete them.
		List<SpecificDateScheduleEntity> specificDateSchedules = specificDateScheduleDao
				.findAllSpecificDateSchedulesByAppId(appId);
		for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateSchedules) {
			// Delete the specific date schedule from database
			deleteSpecificDateSchedule(specificDateScheduleEntity);

			// Ask ScalingJobManager to delete scaling job
			scheduleJobManager.deleteJob(appId, specificDateScheduleEntity.getId(), ScheduleTypeEnum.SPECIFIC_DATE);
		}

		// Get all the recurring schedules for the specifies application id and delete them.
		List<RecurringScheduleEntity> recurringSchedules = recurringScheduleDao.findAllRecurringSchedulesByAppId(appId);
		for (RecurringScheduleEntity recurringScheduleEntity : recurringSchedules) {
			// Delete the recurring date schedule from database
			deleteRecurringSchedule(recurringScheduleEntity);

			// Ask ScalingJobManager to delete scaling job
			scheduleJobManager.deleteJob(appId, recurringScheduleEntity.getId(), ScheduleTypeEnum.RECURRING);
		}

		// Delete all the active schedules for the application
		deleteActiveSchedules(appId);
	}

	private void deleteSpecificDateSchedule(SpecificDateScheduleEntity specificDateScheduleEntity) {
		try {

			specificDateScheduleDao.delete(specificDateScheduleEntity);
		} catch (DatabaseValidationException dve) {
			validationErrorResult.addErrorForDatabaseValidationException(dve, "database.error.delete.failed",
					"app_id=" + specificDateScheduleEntity.getAppId());
			throw new SchedulerInternalException("Database error", dve);
		}
	}

	private void deleteRecurringSchedule(RecurringScheduleEntity recurringScheduleEntity) {
		try {
			recurringScheduleDao.delete(recurringScheduleEntity);
		} catch (DatabaseValidationException dve) {
			validationErrorResult.addErrorForDatabaseValidationException(dve, "database.error.delete.failed",
					"app_id=" + recurringScheduleEntity.getAppId());
			throw new SchedulerInternalException("Database error", dve);
		}
	}

	private void deleteActiveSchedules(String appId) {
		try {
			List<ActiveScheduleEntity> activeScheduleEntities = activeScheduleDao.findByAppId(appId);
			logger.info("Delete active schedules for application: " + appId);
			activeScheduleDao.deleteActiveSchedulesByAppId(appId);
			for (ActiveScheduleEntity activeScheduleEntity : activeScheduleEntities) {

				notifyScalingEngineForDelete(activeScheduleEntity.getAppId(), activeScheduleEntity.getId() );

			}
		} catch (DatabaseValidationException dve) {
			validationErrorResult.addErrorForDatabaseValidationException(dve, "database.error.delete.failed",
					"app_id=" + appId);
			throw new SchedulerInternalException("Database error", dve);
		}
	}
	
	@Transactional
	public SynchronizeResult synchronizeSchedules(){
		int createCount = 0;
		int updateCount = 0;
		int deleteCount = 0;
		Map<String, ApplicationSchedules> policySchedulesMap = new HashMap<String, ApplicationSchedules>();
		Map<String, String> appIdAndGuidMap = new HashMap<String, String>();
		Map<String, String> scheduleAppIdGuidMap = new HashMap<String, String>();
		List<PolicyJsonEntity> policyList = null;
		List recurringScheduleList = null;
		List specificDateScheduleList = null;
		try{
			policyList = policyJsonDao.getAllPolicies();
			recurringScheduleList = recurringScheduleDao.getDistinctAppIdAndGuidList();
			specificDateScheduleList = specificDateScheduleDao.getDistinctAppIdAndGuidList();
		}catch(Exception e){
			logger.error("Failed to synchronize schedules", e);
			throw e;
		}
		
		//create or updated
		if(policyList.size() > 0){
			for(PolicyJsonEntity policy : policyList){
				policySchedulesMap.put(policy.getAppId(), this.parseSchedulesFromPolicy(policy));
				scheduleAppIdGuidMap.put(policy.getAppId(), policy.getGuid());
			}
		}
		
		if(recurringScheduleList.size() > 0){
			for(Object ro : recurringScheduleList){
				Object[] roArray = (Object[])ro;
				appIdAndGuidMap.put((String)(roArray[0]), (String)(roArray[1]));
			}
		}
		
		if(specificDateScheduleList.size() > 0){
			for(Object so : specificDateScheduleList){
				Object[] soArray = (Object[])so;
				appIdAndGuidMap.put((String)(soArray[0]), (String)(soArray[1]));
			}
		}
		
		List<ApplicationSchedules> toCreateScheduleList = new ArrayList<ApplicationSchedules>();
		Set<String> toDeletedAppIds = new HashSet<String>();
		for (String appIdInPolicy : policySchedulesMap.keySet()) {
			if (policySchedulesMap.get(appIdInPolicy).getSchedules() != null
					&& policySchedulesMap.get(appIdInPolicy).getSchedules().hasSchedules()) {
				if (appIdAndGuidMap.get(appIdInPolicy) == null) {
					toCreateScheduleList.add(policySchedulesMap.get(appIdInPolicy));
					createCount++;
					continue;
				} else if (!scheduleAppIdGuidMap.get(appIdInPolicy).equals(appIdAndGuidMap.get(appIdInPolicy))) {
					toCreateScheduleList.add(policySchedulesMap.get(appIdInPolicy));
					toDeletedAppIds.add(appIdInPolicy);
					updateCount++;
					continue;
				}
			}else{
				// there is no schedule in the new policy, so the old schedules of this app should be deleted.
				toDeletedAppIds.add(appIdInPolicy);
				updateCount++;
			}
		}
		
		Set<String> appIdInScheduleSet = new HashSet<String>();
		Set<String> appIdInPolicySet = policySchedulesMap.keySet();
		appIdInScheduleSet.addAll(appIdAndGuidMap.keySet());
		for(String appId : appIdInScheduleSet){
			if(!appIdInPolicySet.contains(appId)){
				toDeletedAppIds.add(appId);
				deleteCount++;
			}
		}
		for(String appId : toDeletedAppIds){
			this.deleteSchedules(appId);
			
		}
		
		for(ApplicationSchedules schedule : toCreateScheduleList){
			this.createSchedules(schedule.getSchedules());
		}		
		
		return new SynchronizeResult(createCount,updateCount,deleteCount);
		
	}
	
	private ApplicationSchedules parseSchedulesFromPolicy(PolicyJsonEntity policyJsonEntity){
		ObjectMapper mapper = new ObjectMapper();
		String policyJson = policyJsonEntity.getPolicyJson();
		ApplicationSchedules applicationSchedules = null;
		mapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
		try {
			applicationSchedules = mapper.readValue(policyJson, ApplicationSchedules.class);
			if(applicationSchedules!=null && applicationSchedules.getSchedules()!=null&&applicationSchedules.getSchedules().hasSchedules()){
				Schedules schedules = applicationSchedules.getSchedules();
				List<RecurringScheduleEntity> recurringSchedules = schedules.getRecurringSchedule();
				List<SpecificDateScheduleEntity> specificDateSchedules = schedules.getSpecificDate();
				if(recurringSchedules != null){
					for(RecurringScheduleEntity recurring : recurringSchedules){
						recurring.setAppId(policyJsonEntity.getAppId());
						recurring.setTimeZone(schedules.getTimeZone());
						recurring.setDefaultInstanceMinCount(applicationSchedules.getInstanceMinCount());
						recurring.setDefaultInstanceMaxCount(applicationSchedules.getInstanceMaxCount());
						recurring.setGuid(policyJsonEntity.getGuid());
					}
				}
				if(specificDateSchedules != null){
					for(SpecificDateScheduleEntity specificDate : specificDateSchedules){
						specificDate.setAppId(policyJsonEntity.getAppId());
						specificDate.setTimeZone(schedules.getTimeZone());
						specificDate.setDefaultInstanceMinCount(applicationSchedules.getInstanceMinCount());
						specificDate.setDefaultInstanceMaxCount(applicationSchedules.getInstanceMaxCount());
						specificDate.setGuid(policyJsonEntity.getGuid());
					}
				}
				 
			}
			
		} catch (IOException e) {
			logger.error("Failed to parse policy, policy_json:" + policyJson, e);
			applicationSchedules = null;
		}
		return applicationSchedules;
	}
	private void notifyScalingEngineForDelete(String appId, long scheduleId) {
		String scalingEnginePathActiveSchedule = ScalingEngineUtil.getScalingEngineActiveSchedulePath(scalingEngineUrl,
				appId, scheduleId);
		String message = messageBundleResourceHelper.lookupMessage("scalingengine.notification.activeschedule.remove",
				appId, scheduleId);
		logger.info(message);
		try {
			restOperations.delete(scalingEnginePathActiveSchedule);
		} catch (HttpStatusCodeException hce) {
			if (hce.getStatusCode() == HttpStatus.NOT_FOUND) {
				message = messageBundleResourceHelper
						.lookupMessage("scalingengine.notification.activeschedule.notFound", appId, scheduleId);
				logger.info(message, hce);
			} else {
				String errorMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.error",
						hce.getResponseBodyAsString(), appId, scheduleId, "delete");
				throw new SchedulerInternalException(errorMessage, hce);
			}
		} catch (ResourceAccessException rae) {
			String errorMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.error",
					rae.getMessage(), appId, scheduleId, "delete");
			throw new SchedulerInternalException(errorMessage, rae);
		}
	}
}
