package org.cloudfoundry.autoscaler.data.couchdb.connection.manager;

import org.ektorp.CouchDbConnector;

public abstract class TypedRepoConnectionManager {
	protected abstract void initRepo(CouchDbConnector couchdbconnector, boolean createIfNotExist);	
}
