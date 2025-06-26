package org.cloudfoundry.autoscaler.scheduler.service;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.text.ParseException;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.TimeZone;
import java.util.stream.Collectors;
import java.util.stream.Stream;
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
import org.cloudfoundry.autoscaler.scheduler.util.ScalingEngineUtil;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.SchedulerInternalException;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.quartz.CronExpression;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.client.HttpStatusCodeException;
import org.springframework.web.client.ResourceAccessException;
import org.springframework.web.client.RestOperations;

/** Service class to persist the schedule entity in the database and create scheduled job. */
@Service
public class ScheduleManager {

  @Autowired private SpecificDateScheduleDao specificDateScheduleDao;
  @Autowired private RecurringScheduleDao recurringScheduleDao;
  @Autowired private ActiveScheduleDao activeScheduleDao;
  @Autowired private PolicyJsonDao policyJsonDao;
  @Autowired private ScheduleJobManager scheduleJobManager;
  @Autowired private RestOperations restOperations;
  @Autowired private ValidationErrorResult validationErrorResult;
  @Autowired private MessageBundleResourceHelper messageBundleResourceHelper;

  @Value("${autoscaler.scalingengine.url}")
  private String scalingEngineUrl;

  private Logger logger = LoggerFactory.getLogger(this.getClass());

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
      allSpecificDateScheduleEntitiesForApp =
          specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId);
      if (!allSpecificDateScheduleEntitiesForApp.isEmpty()) {
        schedules.setSpecificDate(allSpecificDateScheduleEntitiesForApp);
      }

      allRecurringScheduleEntitiesForApp =
          recurringScheduleDao.findAllRecurringSchedulesByAppId(appId);
      if (!allRecurringScheduleEntitiesForApp.isEmpty()) {
        schedules.setRecurringSchedule(allRecurringScheduleEntitiesForApp);
      }

    } catch (DatabaseValidationException dve) {

      validationErrorResult.addErrorForDatabaseValidationException(
          dve, "database.error.get.failed", "app_id=" + appId);
      throw new SchedulerInternalException("Database error", dve);
    }

    return applicationSchedules;
  }

  /**
   * This method calls the helper method to sets up the basic common information in the schedule
   * entities.
   *
   * @param appId
   * @param applicationPolicy
   */
  public void setUpSchedules(String appId, String guid, ApplicationSchedules applicationPolicy) {

    // If there are schedules then only set the meta data in the schedule entities
    if (applicationPolicy.getSchedules() != null
        && applicationPolicy.getSchedules().hasSchedules()) {
      List<SpecificDateScheduleEntity> specificDateSchedules =
          applicationPolicy.getSchedules().getSpecificDate();
      if (specificDateSchedules != null) {
        for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateSchedules) {
          // Sets the meta data in specific date schedules list
          setUpSchedules(appId, guid, applicationPolicy, specificDateScheduleEntity);
        }
      }

      // Call the setUpSchedules to set the meta data in recurring schedules list
      List<RecurringScheduleEntity> recurringSchedules =
          applicationPolicy.getSchedules().getRecurringSchedule();
      if (recurringSchedules != null) {
        for (RecurringScheduleEntity recurringScheduleEntity : recurringSchedules) {
          setUpSchedules(appId, guid, applicationPolicy, recurringScheduleEntity);
        }
      }
    }
  }

  /**
   * Sets the meta data(like the appId, timeZone etc) in the specified entity.
   *
   * @param appId
   * @param applicationPolicy
   * @param scheduleEntity
   */
  private void setUpSchedules(
      String appId,
      String guid,
      ApplicationSchedules applicationPolicy,
      ScheduleEntity scheduleEntity) {
    scheduleEntity.setAppId(appId);
    scheduleEntity.setTimeZone(applicationPolicy.getSchedules().getTimeZone());
    scheduleEntity.setDefaultInstanceMinCount(applicationPolicy.getInstanceMinCount());
    scheduleEntity.setDefaultInstanceMaxCount(applicationPolicy.getInstanceMaxCount());
    scheduleEntity.setGuid(guid);
  }

  /**
   * Calls private helper methods to persist the schedules in the database and calls
   * ScalingJobManager to create scaling action jobs.
   *
   * @param schedules
   */
  @Transactional
  public void createSchedules(Schedules schedules) {
    List<RecurringScheduleEntity> recurringSchedules = schedules.getRecurringSchedule();
    List<SpecificDateScheduleEntity> specificDateSchedules = schedules.getSpecificDate();

    if (recurringSchedules != null) {
      for (RecurringScheduleEntity recurringScheduleEntity : recurringSchedules) {
        // Persist the schedule in database
        RecurringScheduleEntity savedScheduleEntity =
            saveNewRecurringSchedule(recurringScheduleEntity);

        // Ask ScalingJobManager to create scaling job
        if (savedScheduleEntity != null) {
          SpecificDateScheduleEntity compensatorySchedule =
              createCompensatorySchedule(recurringScheduleEntity);
          if (compensatorySchedule != null) {
            // create a compensatory schedule to bring the first fire back
            if (specificDateSchedules == null) {
              specificDateSchedules = new ArrayList<>();
            }
            specificDateSchedules.add(compensatorySchedule);
            logger.debug(
                "add an addition specific date event to compensate the misfire for TODAY: "
                    + compensatorySchedule.toString());
          }
          scheduleJobManager.createCronJob(savedScheduleEntity);
        }
      }
    }

    if (specificDateSchedules != null) {
      for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateSchedules) {
        // Persist the schedule in database
        SpecificDateScheduleEntity savedScheduleEntity =
            saveNewSpecificDateSchedule(specificDateScheduleEntity);

        // Ask ScalingJobManager to create scaling job
        if (savedScheduleEntity != null) {
          scheduleJobManager.createSimpleJob(savedScheduleEntity);
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

      validationErrorResult.addErrorForDatabaseValidationException(
          dve, "database.error.create.failed", "app_id=" + specificDateScheduleEntity.getAppId());
      throw new SchedulerInternalException("Database error", dve);
    }
    return savedScheduleEntity;
  }

  private RecurringScheduleEntity saveNewRecurringSchedule(
      RecurringScheduleEntity recurringScheduleEntity) {
    RecurringScheduleEntity savedScheduleEntity;
    try {
      savedScheduleEntity = recurringScheduleDao.create(recurringScheduleEntity);
    } catch (DatabaseValidationException dve) {
      validationErrorResult.addErrorForDatabaseValidationException(
          dve, "database.error.create.failed", "app_id=" + recurringScheduleEntity.getAppId());
      throw new SchedulerInternalException("Database error", dve);
    }
    return savedScheduleEntity;
  }

  private SpecificDateScheduleEntity createCompensatorySchedule(
      RecurringScheduleEntity recurringScheduleEntity) {

    SpecificDateScheduleEntity compenstatorySchedule = null;
    try {
      ZoneId timezone = ZoneId.of(recurringScheduleEntity.getTimeZone());
      CronExpression expression =
          new CronExpression(
              ScheduleJobHelper.convertRecurringScheduleToCronExpression(
                  recurringScheduleEntity.getStartTime(), recurringScheduleEntity));
      expression.setTimeZone(TimeZone.getTimeZone(timezone));

      Date firstValidTime =
          expression.getNextValidTimeAfter(
              Date.from(LocalDate.now(timezone).atStartOfDay(timezone).toInstant()));
      if (recurringScheduleEntity.getStartTime() == LocalTime.of(0, 0)) {
        firstValidTime =
            expression.getNextValidTimeAfter(
                Date.from(
                    LocalDate.now(timezone).atStartOfDay(timezone).toInstant().minusSeconds(60)));
      }

      if (firstValidTime
          .toInstant()
          .atZone(timezone)
          .toLocalDateTime()
          .isBefore(LocalDateTime.now(timezone))) {
        if (recurringScheduleEntity.getStartDate() == null
            || !(recurringScheduleEntity
                .getStartDate()
                .atStartOfDay(timezone)
                .isAfter(ZonedDateTime.now(timezone)))) {
          compenstatorySchedule = new SpecificDateScheduleEntity();
          compenstatorySchedule.copy(recurringScheduleEntity);
          LocalDateTime startDateTime = LocalDateTime.now(timezone).plusMinutes(1);
          LocalDateTime endDateTime =
              LocalDateTime.of(LocalDate.now(timezone), recurringScheduleEntity.getEndTime());
          compenstatorySchedule.setStartDateTime(startDateTime);
          compenstatorySchedule.setEndDateTime(endDateTime);
        }
      }

    } catch (ParseException e) {
      logger.error("Invalid parse or clone");
    }

    return compenstatorySchedule;
  }

  /**
   * Calls private helper methods to delete the schedules from the database and calls
   * ScalingJobManager to delete scaling action jobs.
   *
   * @param appId
   */
  @Transactional
  public void deleteSchedules(String appId) {

    // Get all the specific date schedules for the specifies application id and delete them.
    List<SpecificDateScheduleEntity> specificDateSchedules =
        specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId);
    for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateSchedules) {
      // Delete the specific date schedule from database
      deleteSpecificDateSchedule(specificDateScheduleEntity);

      // Ask ScalingJobManager to delete scaling job
      scheduleJobManager.deleteJob(
          appId, specificDateScheduleEntity.getId(), ScheduleTypeEnum.SPECIFIC_DATE);
    }

    // Get all the recurring schedules for the specifies application id and delete them.
    List<RecurringScheduleEntity> recurringSchedules =
        recurringScheduleDao.findAllRecurringSchedulesByAppId(appId);
    for (RecurringScheduleEntity recurringScheduleEntity : recurringSchedules) {
      // Delete the recurring date schedule from database
      deleteRecurringSchedule(recurringScheduleEntity);

      // Ask ScalingJobManager to delete scaling job
      scheduleJobManager.deleteJob(
          appId, recurringScheduleEntity.getId(), ScheduleTypeEnum.RECURRING);
    }

    // Delete all the active schedules for the application
    deleteActiveSchedules(appId);
  }

  private void deleteSpecificDateSchedule(SpecificDateScheduleEntity specificDateScheduleEntity) {
    try {

      specificDateScheduleDao.delete(specificDateScheduleEntity);
    } catch (DatabaseValidationException dve) {
      validationErrorResult.addErrorForDatabaseValidationException(
          dve, "database.error.delete.failed", "app_id=" + specificDateScheduleEntity.getAppId());
      throw new SchedulerInternalException("Database error", dve);
    }
  }

  private void deleteRecurringSchedule(RecurringScheduleEntity recurringScheduleEntity) {
    try {
      recurringScheduleDao.delete(recurringScheduleEntity);
    } catch (DatabaseValidationException dve) {
      validationErrorResult.addErrorForDatabaseValidationException(
          dve, "database.error.delete.failed", "app_id=" + recurringScheduleEntity.getAppId());
      throw new SchedulerInternalException("Database error", dve);
    }
  }

  private void deleteActiveSchedules(String appId) {
    try {
      List<ActiveScheduleEntity> activeScheduleEntities = activeScheduleDao.findByAppId(appId);
      logger.info("Delete active schedules for application: " + appId);
      activeScheduleDao.deleteActiveSchedulesByAppId(appId);
      for (ActiveScheduleEntity activeScheduleEntity : activeScheduleEntities) {

        notifyScalingEngineForDelete(activeScheduleEntity.getAppId(), activeScheduleEntity.getId());
      }
    } catch (DatabaseValidationException dve) {
      validationErrorResult.addErrorForDatabaseValidationException(
          dve, "database.error.delete.failed", "app_id=" + appId);
      throw new SchedulerInternalException("Database error", dve);
    }
  }

  @Transactional
  public SynchronizeResult synchronizeSchedules() {
    int createCount = 0;
    int updateCount = 0;
    int deleteCount = 0;
    Map<String, ApplicationSchedules> policySchedulesMap = new HashMap<>();
    Map<String, String> appIdAndGuidMap;
    Map<String, String> scheduleAppIdGuidMap = new HashMap<>();
    List<PolicyJsonEntity> policyList = null;
    Map<String, String> recurringScheduleList = null;
    Map<String, String> specificDateScheduleList = null;
    try {
      policyList = policyJsonDao.getAllPolicies();
      recurringScheduleList = recurringScheduleDao.getDistinctAppIdAndGuidList();
      specificDateScheduleList = specificDateScheduleDao.getDistinctAppIdAndGuidList();
    } catch (Exception e) {
      logger.error("Failed to synchronize schedules", e);
      throw e;
    }

    // create or updated
    if (policyList.size() > 0) {
      for (PolicyJsonEntity policy : policyList) {
        policySchedulesMap.put(policy.getAppId(), this.parseSchedulesFromPolicy(policy));
        scheduleAppIdGuidMap.put(policy.getAppId(), policy.getGuid());
      }
    }

    appIdAndGuidMap =
        Stream.concat(
                recurringScheduleList.entrySet().stream(),
                specificDateScheduleList.entrySet().stream())
            .collect(Collectors.toMap(Map.Entry::getKey, Map.Entry::getValue, (v1, v2) -> v2));

    List<ApplicationSchedules> toCreateScheduleList = new ArrayList<>();
    Set<String> toDeletedAppIds = new HashSet<>();
    for (String appIdInPolicy : policySchedulesMap.keySet()) {
      if (policySchedulesMap.get(appIdInPolicy).getSchedules() != null
          && policySchedulesMap.get(appIdInPolicy).getSchedules().hasSchedules()) {
        if (appIdAndGuidMap.get(appIdInPolicy) == null) {
          toCreateScheduleList.add(policySchedulesMap.get(appIdInPolicy));
          createCount++;
          continue;
        } else if (!scheduleAppIdGuidMap
            .get(appIdInPolicy)
            .equals(appIdAndGuidMap.get(appIdInPolicy))) {
          toCreateScheduleList.add(policySchedulesMap.get(appIdInPolicy));
          toDeletedAppIds.add(appIdInPolicy);
          updateCount++;
          continue;
        }
      } else {
        // there is no schedule in the new policy, so the old schedules of this app should be
        // deleted.
        toDeletedAppIds.add(appIdInPolicy);
        updateCount++;
      }
    }

    Set<String> appIdInScheduleSet = new HashSet<String>();
    Set<String> appIdInPolicySet = policySchedulesMap.keySet();
    appIdInScheduleSet.addAll(appIdAndGuidMap.keySet());
    for (String appId : appIdInScheduleSet) {
      if (!appIdInPolicySet.contains(appId)) {
        toDeletedAppIds.add(appId);
        deleteCount++;
      }
    }
    for (String appId : toDeletedAppIds) {
      this.deleteSchedules(appId);
    }

    for (ApplicationSchedules schedule : toCreateScheduleList) {
      this.createSchedules(schedule.getSchedules());
    }

    return new SynchronizeResult(createCount, updateCount, deleteCount);
  }

  private ApplicationSchedules parseSchedulesFromPolicy(PolicyJsonEntity policyJsonEntity) {
    ObjectMapper mapper = new ObjectMapper();
    String policyJson = policyJsonEntity.getPolicyJson();
    ApplicationSchedules applicationSchedules = null;
    mapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    try {
      applicationSchedules = mapper.readValue(policyJson, ApplicationSchedules.class);
      if (applicationSchedules != null
          && applicationSchedules.getSchedules() != null
          && applicationSchedules.getSchedules().hasSchedules()) {
        Schedules schedules = applicationSchedules.getSchedules();
        List<RecurringScheduleEntity> recurringSchedules = schedules.getRecurringSchedule();
        List<SpecificDateScheduleEntity> specificDateSchedules = schedules.getSpecificDate();
        if (recurringSchedules != null) {
          for (RecurringScheduleEntity recurring : recurringSchedules) {
            recurring.setAppId(policyJsonEntity.getAppId());
            recurring.setTimeZone(schedules.getTimeZone());
            recurring.setDefaultInstanceMinCount(applicationSchedules.getInstanceMinCount());
            recurring.setDefaultInstanceMaxCount(applicationSchedules.getInstanceMaxCount());
            recurring.setGuid(policyJsonEntity.getGuid());
          }
        }
        if (specificDateSchedules != null) {
          for (SpecificDateScheduleEntity specificDate : specificDateSchedules) {
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
    String scalingEnginePathActiveSchedule =
        ScalingEngineUtil.getScalingEngineActiveSchedulePath(scalingEngineUrl, appId, scheduleId);
    String message =
        messageBundleResourceHelper.lookupMessage(
            "scalingengine.notification.activeschedule.remove", appId, scheduleId);
    logger.info(message);
    try {
      restOperations.delete(scalingEnginePathActiveSchedule);
    } catch (HttpStatusCodeException hce) {

      String errorMessage =
          messageBundleResourceHelper.lookupMessage(
              "scalingengine.notification.error",
              hce.getResponseBodyAsString(),
              appId,
              scheduleId,
              "delete");
      throw new SchedulerInternalException(errorMessage, hce);

    } catch (ResourceAccessException rae) {
      String errorMessage =
          messageBundleResourceHelper.lookupMessage(
              "scalingengine.notification.error", rae.getMessage(), appId, scheduleId, "delete");
      throw new SchedulerInternalException(errorMessage, rae);
    }
  }
}
