package org.cloudfoundry.autoscaler.data.couchdb.document;

import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;




@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='ServiceConfig'")
public class ServiceConfig extends TypedCouchDbDocument {

    /**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	@JsonProperty("_id")
    private String serviceId;
    private long persistTimeInDb = 3600;// seconds
    
    public ServiceConfig() {
		super();
	}

	public ServiceConfig(String serviceId, long persistTimeInDb) {
		super();
		this.serviceId = serviceId;
		this.persistTimeInDb = persistTimeInDb;
	}

	public String getServiceId() {
        return serviceId;
    }

    public void setServiceId(String serviceId) {
        this.serviceId = serviceId;
    }

    public long getPersistTimeInDb() {
        return persistTimeInDb;
    }

    public void setPersistTimeInDb(long persistTimeInDB) {
        this.persistTimeInDb = persistTimeInDB;
    }


}

