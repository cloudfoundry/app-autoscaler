package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document;

import org.ektorp.support.CouchDbDocument;

public class TypedCouchDbDocument extends CouchDbDocument {
    protected String type =  this.getClass().getSimpleName();

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }
}
