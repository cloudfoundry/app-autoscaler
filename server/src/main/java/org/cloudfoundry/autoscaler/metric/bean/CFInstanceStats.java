package org.cloudfoundry.autoscaler.metric.bean;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Collections;
import java.util.Date;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;


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

    private static Date parseDate(String dateFormat, String date) {
        try {
            return new SimpleDateFormat(dateFormat).parse(date);
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
            this.time = parseDateBasedOnTimeStamp((String) parse(String.class, attributes.get("time")));
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

        private Date parseDateBasedOnTimeStamp(String rawDate) {
            // The regex in something closer to English:
            // D4-D2-D2[T ]D2:D2:D2{.D9}?TZ...
            Pattern p = Pattern.compile("\\A(\\d{4}-\\d{2}-\\d{2})([ T])(\\d{2}:\\d{2}:\\d{2})(?:(\\.\\d{9})?)(.*)\\z");
            Matcher m = p.matcher(rawDate);
            if (!m.find()) {
                return null;
            }
            String actualDate = m.group(1) + m.group(2) + m.group(3);
            String dateFormat = "yyyy-MM-dd";
            // T or space?
            if (m.group(2).equals("T")) {
                dateFormat += "'T'";
            } else {
                dateFormat += m.group(2);
            }
            dateFormat += "HH:mm:ss";
            // milliseconds?
            if (m.group(4) != null) {
                actualDate += m.group(4).substring(0, 4);
                dateFormat += ".SSS";
            }
            // Do a separate part on the timezone part
            String timezonePart = m.group(5);
            // Handle either "Z" or "\s\d{4}
            if (timezonePart.length() > 0) {
                if (timezonePart.equalsIgnoreCase("Z")) {
                    dateFormat += "'" + timezonePart + "'";
                    actualDate += timezonePart;
                } else {
                    Pattern pz = Pattern.compile("\\A( )?([-+]\\d{4})?\\z");
                    Matcher mz = pz.matcher(timezonePart);
                    if (mz.find()) {
                        if (mz.group(1) != null) {
                            dateFormat += mz.group(1);
                            actualDate += mz.group(1);
                        }
                        if (mz.group(2) != null) {
                            dateFormat += "Z";
                            actualDate += mz.group(2);
                        }
                    }
                }
            }
            return CFInstanceStats.parseDate(dateFormat, actualDate);
        }
    }
}
