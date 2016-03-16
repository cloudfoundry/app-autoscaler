package org.cloudfoundry.autoscaler.data.couchdb.dao.base;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.UUID;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.document.BoundApp;
import org.cloudfoundry.autoscaler.data.couchdb.document.TriggerRecord;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.ViewQuery;
import org.ektorp.support.CouchDbDocument;
import org.ektorp.support.CouchDbRepositorySupport;
import org.ektorp.support.DesignDocument.View;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

public class TypedCouchDbRepositorySupport<T> extends CouchDbRepositorySupport<T> {

	private static final Logger logger = Logger.getLogger(TypedCouchDbRepositorySupport.class);
	private static final ObjectMapper mapper = new ObjectMapper();

	private static Object lock=new Object();

	private String designDocName = null;
	
    public TypedCouchDbRepositorySupport(Class<T> type, CouchDbConnector db, boolean createIfNotExists) {
        super(type, db, createIfNotExists);
        this.designDocName = "_design/" + type.getSimpleName();
    }

    public TypedCouchDbRepositorySupport(Class<T> type, CouchDbConnector db, String designDocName) {
        super(type, db, designDocName);
        this.designDocName = "_design/" + designDocName;
    }

    public String getDesignDocName() {
		return designDocName;
	}

	public void initView() throws Exception {
		initStandardDesignDocument();
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
    
    @Override
    public List<T> queryView (String viewName) {
		String[] input = beforeConnection("QUERY",  new String[]{viewName});
		List<T> returnvalue = null;
		try{
			returnvalue = super.queryView(viewName);
		} catch (Exception e){
			logger.error(e.getMessage());
		}
    	afterConnection(input);
    	return returnvalue;
    }
    
    @Override
    public List<T> queryView (String viewName, ComplexKey key) {
		
		String[] input = beforeConnection("QUERY",  new String[]{viewName, key.toJson().toString()});
		List<T> returnvalue = null;
    	try {
    		returnvalue = super.queryView(viewName, key);
		} catch (Exception e){
			logger.error(e.getMessage());
		}
		
    	afterConnection(input);
    	return returnvalue;
    }
    
    
    @Override
    public List<T> queryView (String viewName, int key) {
    	String[] input = beforeConnection("QUERY",  new String[]{viewName, String.valueOf(key)});
		List<T> returnvalue = null;
		try{
    	 returnvalue = super.queryView(viewName, key);
		} catch (Exception e){
			logger.error(e.getMessage());
		}
   	   afterConnection(input);
    	return returnvalue;
    }
    
    
    @Override
    public List<T> queryView (String viewName, String key) {
    	String[] input = beforeConnection("QUERY",  new String[]{viewName, key});
		List<T> returnvalue = null;
		try{
			returnvalue = super.queryView(viewName, key);
		} catch (Exception e){
			logger.error(e.getMessage());
		}
		afterConnection(input);
    	return returnvalue;
    }
    
    
    @Override
    public T get (String id) {
		String[] input = beforeConnection("GET",  new String[]{id});
		T returnvalue = null;
		try{
    	 returnvalue = super.get(id);
		} catch (Exception e){
			logger.error(e.getMessage());
		}

		afterConnection(input);
    	return returnvalue;
    }
    
    
    @Override
    public void add (T entity) {
    	
    	String mapStr = null;
    	try {
			mapStr=mapper.writeValueAsString(entity);
		} catch (JsonProcessingException e1) {
		}
    	
		String[] input = beforeConnection("ADD",  new String[]{mapStr});
    	try {
    		super.add(entity);
		} catch (Exception e){
			logger.error(e.getMessage());
		}
		afterConnection(input);

    }
    
    @Override
    public void update (T entity) {

    	String mapStr = null;
    	try {
			mapStr=mapper.writeValueAsString(entity);
		} catch (JsonProcessingException e1) {
			e1.printStackTrace();
		}

    	
    	String[] input = beforeConnection("UPDATE", new String[]{mapStr});
    	try {
    		super.update(entity);
		} catch (Exception e){
			logger.error(e.getMessage(), e);
		}
		afterConnection(input);

    }
    
    @Override
    public void remove (T entity) {
    	String mapStr = null;
    	try {
			mapStr=mapper.writeValueAsString(entity);
		} catch (JsonProcessingException e1) {
			e1.printStackTrace();
		}
    	
		String[] input = beforeConnection("REMOVE", new String[]{mapStr});    	
    	try {
    		super.remove(entity);
		} catch (Exception e){
			logger.error(e.getMessage(), e);
		}
		afterConnection(input);
    }
    
    public String[] beforeConnection (String httpMethod, String[] args) {

		String uuid = UUID.randomUUID().toString();
		
    	StringBuilder logs = new StringBuilder().append(httpMethod).append(" ").append(" ").append(this.designDocName);
    	for (String arg : args){
    		logs.append(" ").append(arg);
    	}
    	String loggingInfo = logs.toString();
    	
    	logger.debug(new StringBuilder().append(uuid).append(" ").append(loggingInfo));
    	synchronized (lock){
    		CouchDBConnectionProfile.getInstance().increaseConnectionCount(uuid);
    	}
    	
    	return new String[]{uuid, String.valueOf(System.currentTimeMillis()), loggingInfo};
    }
    
    public void afterConnection (String[] args) {
    	
    	synchronized (lock){
    		CouchDBConnectionProfile.getInstance().decreaseConnectionCount(args[0]);
    	}
    	long latency = System.currentTimeMillis() - Long.parseLong(args[1]); 
    	StringBuilder logs = new StringBuilder().append(args[0]).append(" ").append(latency).append(" ").append(args[2]);
    	logger.debug(logs.toString());
    }    
    
    //only for mergeDB task
    public void importRecords(List<T> records) throws Exception {
        for (T source : records) {
                try {
                	((CouchDbDocument) source).setRevision(null);
                    update(source);
                } catch (org.ektorp.UpdateConflictException e) {
                	logger.error("Error:" , e);
                }
            }
    }
    
    //only for mergeDB task
    public void importRecordsWihtServerName(List<T> records, String serverName) throws Exception {
        for (T source : records) {
                try {
                	((CouchDbDocument) source).setRevision(null);
                	if (source.getClass().getSimpleName().equalsIgnoreCase("TriggerRecord")){
                		if( ((TriggerRecord) source).getServerName() == null )
                				((TriggerRecord) source).setServerName(serverName);
                		
                	}
                	else if (source.getClass().getSimpleName().equalsIgnoreCase("BoundApp")){
                		if (((BoundApp) source).getServerName() == null) 
                			((BoundApp) source).setServerName(serverName);
                	}
                	update(source);
                } catch (org.ektorp.UpdateConflictException e) {
                	logger.error("Error:" , e);
                }
            }
    }
    
}
