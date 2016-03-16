package org.cloudfoundry.autoscaler.bean;

import java.io.Serializable;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public class Metric implements Serializable, Cloneable {
    /**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	private String name;
    private String value;
    private String category;
    private String group;
    private long timestamp;

    private String unit;
    private String desc;

    public Metric() {

    }

    public Metric(String name, String value) {
        this.name = name;
        this.value = value;
    }

    public Metric(String name, String value, String category, String group, long timestamp, String unit, String desc) {
        super();
        this.name = name;
        this.value = value;
        this.category = category;
        this.group = group;
        this.timestamp = timestamp;
        this.unit = unit;
        this.desc = desc;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getValue() {
        return value;
    }

    public void setValue(String value) {
        this.value = value;
    }

    public String getCategory() {
        return category;
    }

    public void setCategory(String category) {
        this.category = category;
    }

    public String getGroup() {
        return group;
    }

    public void setGroup(String group) {
        this.group = group;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }

    public String getUnit() {
        return unit;
    }

    public void setUnit(String unit) {
        this.unit = unit;
    }

    public String getDesc() {
        return desc;
    }

    public void setDesc(String desc) {
        this.desc = desc;
    }

    @JsonIgnore
    public String getCompoundName() {
        StringBuilder sb = new StringBuilder();
        sb.append(this.category).append("#");
        sb.append(this.group);
        if (!this.group.equals(this.name)) {
            sb.append("#").append(this.name);
        }
        return sb.toString().toLowerCase();
    }

    public Metric clone() {
        try {
            return (Metric) super.clone();
        } catch (CloneNotSupportedException e) {
            throw new RuntimeException(e);
        }
    }

    public String toString() {
        StringBuilder sb = new StringBuilder("[");
        sb.append("category=").append(category).append(",");
        sb.append("group=").append(group).append(",");
        sb.append("name=").append(name).append(",");
        sb.append("value=").append(value).append(",");
        sb.append("]");
        return sb.toString();
    }
}
