package org.cloudfoundry.autoscaler.servicebroker.data.entity;

import org.ektorp.support.CouchDbDocument;

public class TypedCouchDbDocument extends CouchDbDocument {

	private static final long serialVersionUID = 1L;
	protected String type;

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }
}
