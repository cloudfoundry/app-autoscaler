package org.cloudfoundry.autoscaler.cloudservice.couchdb.connection;

import org.ektorp.CouchDbConnector;

public abstract class TypedRepoConnectionManager {
	protected abstract void initRepo(CouchDbConnector couchdbconnector, boolean createIfNotExist);	
}
