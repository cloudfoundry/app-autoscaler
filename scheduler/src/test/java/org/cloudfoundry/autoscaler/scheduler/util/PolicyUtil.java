package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;

import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;

public class PolicyUtil {

	public static String getPolicyJsonContent() {
		BufferedReader br = new BufferedReader(
				new InputStreamReader(ApplicationSchedules.class.getResourceAsStream("/fakePolicy.json")));
		String tmp;
		String jsonPolicyStr = "";
		try {
			while ((tmp = br.readLine()) != null) {
				jsonPolicyStr += tmp;
			}
		} catch (IOException e) {
			e.printStackTrace();
		}
		jsonPolicyStr = jsonPolicyStr.replaceAll("\\s+", " ");
		return jsonPolicyStr;
	}
}
