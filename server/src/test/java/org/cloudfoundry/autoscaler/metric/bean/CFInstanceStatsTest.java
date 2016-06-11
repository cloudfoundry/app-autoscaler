package org.cloudfoundry.autoscaler.metric.bean;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import java.util.HashMap;
import java.util.Map;
import java.util.Set;

import org.json.JSONObject;
import org.junit.Test;

import org.cloudfoundry.autoscaler.metric.bean.CloudAppInstance;
import org.cloudfoundry.autoscaler.metric.bean.CFInstanceStats;

public class CFInstanceStatsTest {
    private Map<String, Map<String, Object>> getAppInstanceData(String timestamp) {
        String response = "{\"0\":{\"state\":\"RUNNING\",\"stats\":{\"name\":\"slattery\",\"uris\":[\"slattery.192.168.77.77.nip.io\"],\"host\":\"172.20.10.22\",\"port\":60002,\"net_info\":{\"address\":\"172.20.10.22\",\"ports\":[{\"container_port\":8080,\"host_port\":60002},{\"container_port\":2222,\"host_port\":60003}]},\"uptime\":6405,\"mem_quota\":1073741824,\"disk_quota\":1073741824,\"fds_quota\":16384,\"usage\":{\"time\":\"2016-06-09T18:09:15.403922371Z\",\"cpu\":0.000334831818019757,\"mem\":693276672,\"disk\":79220736}}}}";
        JSONObject jsonObj = new JSONObject(response);
        Set<String> keySet = jsonObj.keySet();
        Map<String, Map<String, Object>> instanceStats = new HashMap<String, Map<String, Object>>(keySet.size());
        //logger.debug(String.format("%d instances for app %s", keySet.size(), appId));
        for (String key : keySet) {
            String id = key;
            JSONObject jsonStats = (JSONObject) jsonObj.get(key);

            Map<String, Object> attributes = new HashMap<String, Object>();

            attributes.put("state", "RUNNING");

            Map<String, Object> statsMap = new HashMap<String, Object>();
            JSONObject statsObj = (JSONObject) jsonStats.get("stats");
            statsMap.put("name", statsObj.get("name"));
            statsMap.put("host", statsObj.get("host"));
            statsMap.put("port", statsObj.get("port"));
            statsMap.put("uptime", Double.parseDouble(statsObj.get("uptime").toString()));
            statsMap.put("mem_quota", statsObj.get("mem_quota"));
            statsMap.put("disk_quota", statsObj.get("disk_quota"));
            statsMap.put("fds_quota", statsObj.get("fds_quota"));

            Map<String, Object> usageMap = new HashMap<String, Object>();
            JSONObject usageObj = (JSONObject) statsObj.get("usage");
            usageMap.put("time", timestamp);
            usageMap.put("cpu", Double.parseDouble(usageObj.get("cpu").toString()));
            usageMap.put("mem", Double.parseDouble(usageObj.get("mem").toString()));
            usageMap.put("disk", Integer.parseInt(usageObj.get("disk").toString()));

            statsMap.put("usage", usageMap);

            attributes.put("stats", statsMap);

            instanceStats.put(id, attributes);
        }
        return instanceStats;
    }
    
    @Test
    public void testDEATimestamp() {
        Map<String, Map<String, Object>> stats = getAppInstanceData("1973-04-24 12:03:45 -0700");
        CFInstanceStats cfStat = new CFInstanceStats("0", stats.get("0"));
        CFInstanceStats.Usage instUsage = cfStat.getUsage();
        assertNotNull(instUsage);
        long timestamp = instUsage.getTime().getTime();
        assertEquals(timestamp, 104526225000L);
    }
    
    @Test
    public void testDiegoTimestamp() {
        Map<String, Map<String, Object>> stats = getAppInstanceData("1973-04-24T12:03:45.123456789Z");
        CFInstanceStats cfStat = new CFInstanceStats("0", stats.get("0"));
        CFInstanceStats.Usage instUsage = cfStat.getUsage();
        assertNotNull(instUsage);
        long timestamp = instUsage.getTime().getTime();
        assertEquals(timestamp, 104501025123L);
    }
    
    @Test
    public void testDiegoTimestampWithSpaceTimezone() {
        Map<String, Map<String, Object>> stats = getAppInstanceData("1973-04-24T12:03:45.123456789 -0700");
        CFInstanceStats cfStat = new CFInstanceStats("0", stats.get("0"));
        CFInstanceStats.Usage instUsage = cfStat.getUsage();
        assertNotNull(instUsage);
        long timestamp = instUsage.getTime().getTime();
        assertEquals(timestamp, 104526225123L);
    }
    
    // Don't run this test, as it appears to be run in the locale's local timezone
    // Will only pass when run in northern-hemisphere Pacific Time
    // public void testDiegoTimestampWithNoTimezone() {
    //     Map<String, Map<String, Object>> stats = getAppInstanceData("1973-04-24T12:03:45.123456789");
    //     CFInstanceStats cfStat = new CFInstanceStats("0", stats.get("0"));
    //     CFInstanceStats.Usage instUsage = cfStat.getUsage();
    //     assertNotNull(instUsage);
    //     long timestamp = instUsage.getTime().getTime();
    //     assertEquals(timestamp, 104526225123L);
    // }
}
