package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;

public class PolicyUtil {

  public static String getPolicyJsonContent() throws IOException {
    try (BufferedReader br =
        new BufferedReader(
            new InputStreamReader(
                ApplicationSchedules.class.getResourceAsStream("/fakePolicy.json")))) {
      String tmp;
      String jsonPolicyStr = "";
      while ((tmp = br.readLine()) != null) {
        jsonPolicyStr += tmp;
      }
      jsonPolicyStr = jsonPolicyStr.replaceAll("\\s+", " ");
      return jsonPolicyStr;
    }
  }
}
