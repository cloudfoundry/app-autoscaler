package org.cloudfoundry.autoscaler.data.couchdb.document;

import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;

@JsonIgnoreProperties(ignoreUnknown = true)
@JsonSerialize(include = JsonSerialize.Inclusion.NON_NULL)
@TypeDiscriminator ("doc.type=='MetricDBSegment'")
public class MetricDBSegment extends TypedCouchDbDocument implements Comparable  {
    /**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	private long startTimestamp;
    private long endTimestamp;
    private String metricDBPostfix;
    private int segmentSeq;  
    private String serverName;
    
    public MetricDBSegment() {
    	super();
    }

	public long getStartTimestamp() {
		return startTimestamp;
	}

	public void setStartTimestamp(long startTimestamp) {
		this.startTimestamp = startTimestamp;
	}

	public long getEndTimestamp() {
		return endTimestamp;
	}

	public void setEndTimestamp(long endTimestamp) {
		this.endTimestamp = endTimestamp;
	}

	public String getMetricDBPostfix() {
		return metricDBPostfix;
	}

	public void setMetricDBPostfix(String metricDBPostfix) {
		this.metricDBPostfix = metricDBPostfix;
	}

	public int getSegmentSeq() {
		return segmentSeq;
	}

	public void setSegmentSeq(int segmentSeq) {
		this.segmentSeq = segmentSeq;
	}

	public String getServerName() {
		return serverName;
	}

	public void setServerName(String serverName) {
		this.serverName = serverName;
	}

	@Override
	public int compareTo(Object o) {
		if (this.startTimestamp > ((MetricDBSegment) o).getStartTimestamp())
			return 1;
		if (this.startTimestamp < ((MetricDBSegment) o).getStartTimestamp())
			return -1;
		return 0;
		
	}

	@Override
	public String toString() {
		return new StringBuilder().append(this.getMetricDBPostfix()).append(":")
				.append(this.getServerName()).append(":")
				.append(this.getStartTimestamp()).append(":")
				.append(this.getEndTimestamp()).toString();
	}


	

	
 
}
