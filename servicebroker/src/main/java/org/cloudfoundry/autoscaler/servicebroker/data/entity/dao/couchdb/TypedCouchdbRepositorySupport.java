package org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.couchdb;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Set;

import org.ektorp.CouchDbConnector;
import org.ektorp.ViewQuery;
import org.ektorp.support.CouchDbRepositorySupport;
import org.ektorp.support.DesignDocument.View;

public class TypedCouchdbRepositorySupport<T> extends CouchDbRepositorySupport<T> {

	private String designDocName = null;
	
    public TypedCouchdbRepositorySupport(Class<T> type, CouchDbConnector db, boolean createIfNotExists) {
        super(type, db, createIfNotExists);
        this.designDocName = "_design/" + type.getSimpleName();
    }

    public TypedCouchdbRepositorySupport(Class<T> type, CouchDbConnector db, String designDocName) {
        super(type, db, designDocName);
        this.designDocName = "_design/" + designDocName;
    }

    public String getDesignDocName() {
		return designDocName;
	}
    
    public List<String> getViewNames() throws Exception {
    	
    	Map<String, View> views = this.getDesignDocumentFactory().getFromDatabase(this.db, this.designDocName).getViews();
    	List<String> viewNames = new ArrayList<String>();
    	
    	Set<Map.Entry<String, View>> entryseSet=views.entrySet();  
    	  for (Map.Entry<String, View> entry:entryseSet) {  
    	     viewNames.add(entry.getKey());
    	  }  
        return viewNames;
    }

    public String getViewMeta(String viewName) throws Exception {
    	ViewQuery q = createQuery(viewName).limit(0);    	
        return db.queryView(q).toString();
    }
    
}
