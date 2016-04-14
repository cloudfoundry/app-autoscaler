package org.cloudfoundry.autoscaler.api.validation;

import javax.validation.constraints.NotNull;

public class Metric {
	@NotNull(message="{Metric.name.NotNull}")
    private String name;

	@NotNull(message="{Metric.value.NotNull}")
    private String value;

	@NotNull(message="{Metric.category.NotNull}")
    private String category;

	@NotNull(message="{Metric.group.NotNull}")
    private String group;

	@NotNull(message="{Metric.timestamp.NotNull}")
    private long timestamp;

	@NotNull(message="{Metric.unit.NotNull}")
    private String unit;

    private String desc="";

    public String getName() {
    	return this.name;
    }

    public void setName(String name) {
    	this.name = name;
    }

    public String getValue() {
    	return this.value;
    }

    public void setValue(String value) {
    	this.value = value;
    }

    public String getCategory() {
    	return this.category;
    }

    public void setCategory(String category) {
    	this.category = category;
    }

    public String getGroup() {
    	return this.group;
    }

    public void setGroup(String group) {
    	this.group = group;
    }

    public long getTimestamp() {
    	return this.timestamp;
    }

    public void setTimestamp(long timestamp) {
    	this.timestamp = timestamp;
    }

    public String getUnit() {
    	return this.unit;
    }

    public void setUnit(String unit) {
    	this.unit = unit;
    }

    public String getDesc() {
    	return this.desc;
    }

    public void setDesc(String desc) {
    	this.desc = desc;
    }
}