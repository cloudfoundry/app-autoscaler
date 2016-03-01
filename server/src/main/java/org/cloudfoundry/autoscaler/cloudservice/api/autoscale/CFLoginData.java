package org.cloudfoundry.autoscaler.cloudservice.api.autoscale;

public class CFLoginData {

	private String user;
	private String pass;
	private String target;
	
	public CFLoginData() {
	}
	
	public CFLoginData(String usr, String pas, String tar) {
		user   = usr;
		pass   = pas;
		target = tar;
	}
	
	public String getUser() {
		return user;
	}
	public void setUser(String user) {
		this.user = user;
	}
	public String getPass() {
		return pass;
	}
	public void setPass(String pass) {
		this.pass = pass;
	}
	public String getTarget() {
		return target;
	}
	public void setTarget(String target) {
		this.target = target;
	}
	
}
