package org.cloudfoundry.autoscaler.api.util;

import java.text.MessageFormat;
import java.util.Locale;
import java.util.ResourceBundle;

/**
 * Resouce bundle messages util
 * 
 * 
 * 
 */
public class MessageUtil {

	private final static String BUNDLE_NAME="org.cloudfoundry.autoscaler.api.nls.APIServerMessages";

	private final static ResourceBundle NONLS_MESSAGE_RESOURCE_BUNDLE = ResourceBundle
            .getBundle(BUNDLE_NAME); //$NON-NLS-1$

	
	public final static String RestResponseErrorMsg_build_JSON_error = "RestResponseErrorMsg_build_JSON_error";
	public final static String RestResponseErrorMsg_parse_JSON_error = "RestResponseErrorMsg_parse_JSON_error";
	public final static String RestResponseErrorMsg_config_exist_error = "RestResponseErrorMsg_config_exist_error";
	public final static String RestResponseErrorMsg_policy_not_found_error = "RestResponseErrorMsg_policy_not_found_error";
	public final static String RestResponseErrorMsg_applist_not_empty_error = "RestResponseErrorMsg_applist_not_empty_error";
	public final static String RestResponseErrorMsg_app_not_found_error = "RestResponseErrorMsg_app_not_found_error";
	public final static String RestResponseErrorMsg_cloud_error = "RestResponseErrorMsg_cloud_error";
	public final static String RestResponseErrorMsg_metric_not_supported_error = "RestResponseErrorMsg_metric_not_supported_error";
	public final static String RestResponseErrorMsg_database_error = "RestResponseErrorMsg_database_error";
	public final static String RestResponseErrorMsg_no_attached_policy_error = "RestResponseErrorMsg_no_attached_policy_error";
	public final static String RestResponseErrorMsg_call_bss_fail_error = "RestResponseErrorMsg_call_bss_fail_error";
	public final static String RestResponseErrorMsg_app_info_not_found_error = "RestResponseErrorMsg_app_info_not_found_error";
	public final static String RestResponseErrorMsg_service_not_found_error = "RestResponseErrorMsg_service_not_found_error";
	public final static String RestResponseErrorMsg_internal_server_error = "RestResponseErrorMsg_internal_server_error";
	public final static String RestResponseErrorMsg_input_json_parse_error = "RestResponseErrorMsg_input_json_parse_error";
	public final static String RestResponseErrorMsg_input_json_format_error = "RestResponseErrorMsg_input_json_format_error";
	public final static String RestResponseErrorMsg_input_json_format_location_error = "RestResponseErrorMsg_input_json_format_location_error";
	public final static String RestResponseErrorMsg_output_json_parse_error = "RestResponseErrorMsg_output_json_parse_error";
	public final static String RestResponseErrorMsg_output_json_format_error = "RestResponseErrorMsg_output_json_format_error";
	public final static String RestResponseErrorMsg_policy_not_exist_error = "RestResponseErrorMsg_policy_not_exist_error";
	public final static String RestResponseErrorMsg_internal_authentication_failed_error = "RestResponseErrorMsg_internal_authentication_failed_error";
	
	public final static String RestResponseErrorMsg_retrieve_application_service_information_context = "RestResponseErrorMsg_retrieve_application_service_information_context";
	public final static String RestResponseErrorMsg_retrieve_org_sapce_information_context = "RestResponseErrorMsg_retrieve_org_sapce_information_context";
	public final static String RestResponseErrorMsg_parse_input_json_context = "RestResponseErrorMsg_parse_input_json_context";
	public final static String RestResponseErrorMsg_Create_Update_Policy_context = "RestResponseErrorMsg_Create_Update_Policy_context";
	public final static String RestResponseErrorMsg_Enable_Policy_context = "RestResponseErrorMsg_Enable_Policy_context";
	public final static String RestResponseErrorMsg_Get_Policy_context = "RestResponseErrorMsg_Get_Policy_context";
	public final static String RestResponseErrorMsg_Get_Scaling_History_context = "RestResponseErrorMsg_Get_Scaling_History_context";
	public final static String RestResponseErrorMsg_Get_Metric_Data_context = "RestResponseErrorMsg_Get_Metric_Data_context";
	public final static String RestResponseErrorMsg_update_policy_in_Create_Policy_context = "RestResponseErrorMsg_Update_policy_in_Create_Policy_context";
	public final static String RestResponseErrorMsg_create_policy_in_Create_Policy_context = "RestResponseErrorMsg_create_policy_in_Create_Policy_context";
	public final static String RestResponseErrorMsg_attach_policy_in_Create_Policy_context = "RestResponseErrorMsg_attach_policy_in_Create_Policy_context";
	public final static String RestResponseErrorMsg_detach_policy_in_Delete_Policy_context = "RestResponseErrorMsg_detach_policy_in_Delete_Policy_context";
	public final static String RestResponseErrorMsg_delete_policy_in_Delete_Policy_context = "RestResponseErrorMsg_delete_policy_in_Delete_Policy_context";
	public final static String RestResponseErrorMsg_get_policy_in_Get_Policy_context = "RestResponseErrorMsg_get_policy_in_Get_Policy_context";
	public final static String RestResponseErrorMsg_enable_policy_in_Enable_Policy_context = "RestResponseErrorMsg_enable_policy_in_Enable_Policy_context";
	public final static String RestResponseErrorMsg_get_history_in_Get_History_context = "RestResponseErrorMsg_get_history_in_Get_History_context";
	public final static String RestResponseErrorMsg_get_metric_in_Get_Metric_context = "RestResponseErrorMsg_get_metric_in_Get_Metric_context";
	public final static String RestResponseErrorMsg_API_Server_retrieve_service_information_context = "RestResponseErrorMsg_API_Server_retrieve_service_information_context";
	
	
    /**
     * Gets messages
     * 
     * @param key
     * @return
     */
	public static String getMessageString(String key) {
        return NONLS_MESSAGE_RESOURCE_BUNDLE.getString(key);
    }

    /**
     * Get messages
     * 
     * @param key
     * @param params
     * @return
     */
    public static String getMessageString(String key, Object... params) {
        if (params == null || params.length == 0) {
            getMessageString(key);
        }
        return MessageFormat.format(NONLS_MESSAGE_RESOURCE_BUNDLE.getString(key), params);
    }
    
    /**
     * Gets messages
     * 
     * @param key
     * @return
     */
	public static String getMessageString(String key, Locale locale) {
    	ResourceBundle MESSAGE_RESOURCE_BUNDLE = ResourceBundle
                .getBundle(BUNDLE_NAME, locale); //$NON-NLS-1$
        return MESSAGE_RESOURCE_BUNDLE.getString(key);
    } 
    
    /**
     * Get messages
     * 
     * @param key
     * @param locale
     * @param params
     * @return
     */
    public static String getMessageString(String key, Locale locale, Object... params) {
        if (params == null || params.length == 0) {
            return getMessageString(key,locale);
        }  
    	ResourceBundle MESSAGE_RESOURCE_BUNDLE = ResourceBundle
                .getBundle(BUNDLE_NAME, locale); //$NON-NLS-1$
        return MessageFormat.format(MESSAGE_RESOURCE_BUNDLE.getString(key), params);
    }

}
