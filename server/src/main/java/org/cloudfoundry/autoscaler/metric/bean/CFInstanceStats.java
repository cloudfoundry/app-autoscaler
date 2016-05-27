package org.cloudfoundry.autoscaler.metric.bean;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Collections;
import java.util.Date;
import java.util.List;
import java.util.Locale;
import java.util.Map;



@SuppressWarnings({ "rawtypes", "unchecked" })
public class CFInstanceStats {
    private int cores;
    private long diskQuota;
    private int fdsQuota;
    private String host;
    private String id;
    private long memQuota;
    private String name;
    private int port;
    private InstanceState state;
    private double uptime;
    private List<String> uris;
    private Usage usage;

    public CFInstanceStats(String id, Map<String, Object> attributes) {
        this.id = id;
        String instanceState = (String) parse(String.class, attributes.get("state"));
        this.state = InstanceState.valueOfWithDefault(instanceState);
        Map stats = (Map) parse(Map.class, attributes.get("stats"));
        if (stats != null) {
            this.cores = ((Integer) parse(Integer.class, stats.get("cores"))).intValue();
            this.name = ((String) parse(String.class, stats.get("name")));
            Map usageValue = (Map) parse(Map.class, stats.get("usage"));

            if (usageValue != null) {
                this.usage = new Usage(usageValue);
            }
            this.diskQuota = ((Long) parse(Long.class, stats.get("disk_quota"))).longValue();
            this.port = ((Integer) parse(Integer.class, stats.get("port"))).intValue();
            this.memQuota = ((Long) parse(Long.class, stats.get("mem_quota"))).longValue();
            List statsValue = (List) parse(List.class, stats.get("uris"));
            if (statsValue != null) {
                this.uris = Collections.unmodifiableList(statsValue);
            }
            this.fdsQuota = ((Integer) parse(Integer.class, stats.get("fds_quota"))).intValue();
            this.host = ((String) parse(String.class, stats.get("host")));
            this.uptime = ((Double) parse(Double.class, stats.get("uptime"))).doubleValue();
        }
    }

    private static <T> T parse(Class<T> clazz, Object object) {
        Object defaultValue = null;
        try {
            if (clazz == Date.class) {
                String stringValue = (String) parse(String.class, object);
                return clazz.cast(new SimpleDateFormat("EEE MMM d HH:mm:ss Z yyyy", Locale.US).parse(stringValue));
            }

            if (clazz == Integer.class)
                defaultValue = 0;
            else if (clazz == Long.class)
                defaultValue = 0L;
            else if (clazz == Double.class) {
                defaultValue = 0.0;
            }

            if (object == null) {
                return (T) defaultValue;
            }

            if (clazz == Integer.class) {
                return clazz.cast(Integer.valueOf(((Number) object).intValue()));
            }
            if (clazz == Long.class) {
                return clazz.cast(Long.valueOf(((Number) object).longValue()));
            }

            return clazz.cast(object);
        } catch (ClassCastException e) {
        } catch (ParseException e) {
        }
        return (T) defaultValue;
    }

    private static Date parseDate(String date) {
        try {
            return new SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss.SSS").parse(date);
        } catch (ParseException e) {
        }
        return null;
    }

    public int getCores() {
        return this.cores;
    }

    public long getDiskQuota() {
        return this.diskQuota;
    }

    public int getFdsQuota() {
        return this.fdsQuota;
    }

    public String getHost() {
        return this.host;
    }

    public String getId() {
        return this.id;
    }

    public long getMemQuota() {
        return this.memQuota;
    }

    public String getName() {
        return this.name;
    }

    public int getPort() {
        return this.port;
    }

    public InstanceState getState() {
        return this.state;
    }

    public double getUptime() {
        return this.uptime;
    }

    public List<String> getUris() {
        return this.uris;
    }

    public Usage getUsage() {
        return this.usage;
    }

    public static class Usage {
        private double cpu;
        private int disk;
        private double mem;
        private Date time;

        public Usage(Map<String, Object> attributes) {
            String rawTime = (String) parse(String.class, attributes.get("time"));
            int endPoint = rawTime.indexOf(".") + 4;
            String adjustedTime = endPoint > 3 ? rawTime.substring(0, endPoint) : rawTime;
            this.time = CFInstanceStats.parseDate(adjustedTime);
            this.cpu = ((Double) parse(Double.class, attributes.get("cpu"))).doubleValue();
            this.disk = ((Integer) parse(Integer.class, attributes.get("disk"))).intValue();
            this.mem = ((Double) parse(Double.class, attributes.get("mem"))).doubleValue();
        }

        public double getCpu() {
            return this.cpu;
        }

        public int getDisk() {
            return this.disk;
        }

        public double getMem() {
            return this.mem;
        }

        public Date getTime() {
            return (Date)this.time.clone();
        }
    }
}
