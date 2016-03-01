package org.cloudfoundry.autoscaler.api.util;

import java.util.HashSet;
import java.util.Locale;
import java.util.Set;
import java.util.regex.Pattern;

import javax.servlet.http.HttpServletRequest;

import org.apache.log4j.Logger;

public class LocaleUtil {
	private static final Logger logger = Logger.getLogger(LocaleUtil.class);
	private static Set<String> supportedLanguage = new HashSet<String>();
    static {
    	supportedLanguage.add("zh");
    	supportedLanguage.add("pt");
    	supportedLanguage.add("pl");
    	supportedLanguage.add("ko");
    	supportedLanguage.add("ja");
    	supportedLanguage.add("it");
    	supportedLanguage.add("fr");
    	supportedLanguage.add("es");
    	supportedLanguage.add("de");
    	supportedLanguage.add("en");
    }
    
	private final static Locale DEFAULT_LOCALE = Locale.US;
	private static final Pattern languagePattern=Pattern.compile("([a-z]{2}[[-[A-Z]{2}]?[;q=[0.[0-9]]|1]?])*,");
	private static final Pattern qPattern=Pattern.compile("[;q=[0.[0-9]]|1]");
	private static final Pattern localePattern=Pattern.compile("[a-z]{2}-[A-Z]{2}");
	
    public static Locale getLocale(HttpServletRequest httpServletRequest) {
 
    	Locale locale = httpServletRequest.getLocale();
    	logger.info(">>>>>>>>> Locale from httpServletRequest is: " + locale.toString());
    	if (locale.toString().isEmpty() || locale.toString().equalsIgnoreCase("null")){
    		try {
    			logger.info(">>>>>>>>> Fail to get locale from httpServletRequest and try to get from Http Header");
    			String acceptLanguageHeader = httpServletRequest.getHeader("Accept-Language");

    			String[] acceptLanguages=languagePattern.split(acceptLanguageHeader);
    			for (String acceptLanguage : acceptLanguages) {
    				acceptLanguage = qPattern.matcher(acceptLanguage).replaceAll("").trim();
    				if (localePattern.matcher(acceptLanguage).matches()){
    					String [] languageAndCountry = acceptLanguage.split("-");
    					String language = languageAndCountry[0].trim();
    					String country = languageAndCountry[1].trim();
    					if (supportedLanguage.contains(language)){
    						locale = new Locale(language, country, "");
    						break;
    					}
    				} else {
    					if (supportedLanguage.contains(acceptLanguage)){
    						locale = new Locale(acceptLanguage);
    						break;
    					}
    				}
    			}
    		} catch (Exception e) {
    			logger.info(">>>>>>>>> Exception happended and return DEFAULT_LOCALE");
    	    	return DEFAULT_LOCALE;
    		}
    	} 
    	logger.info(">>>>>>>>> Returned Locale is: " + locale.toString());
    	return locale;
    	
    }
    
    
   
    
}
