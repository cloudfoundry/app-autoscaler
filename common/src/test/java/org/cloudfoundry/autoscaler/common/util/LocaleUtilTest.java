package org.cloudfoundry.autoscaler.common.util;

import java.util.Locale;
import static org.junit.Assert.assertEquals;

import org.springframework.mock.web.MockHttpServletRequest;
import org.cloudfoundry.autoscaler.common.util.LocaleUtil;
import org.junit.Test;

public class LocaleUtilTest {

	
	@Test
	public void getDefaultLocaleTest() {
		MockHttpServletRequest request = new MockHttpServletRequest();
	    Locale locale_empty = LocaleUtil.getLocale(request);
	    Locale default_locale = Locale.forLanguageTag("en");
		assertEquals(default_locale, locale_empty);
	}
	
	@Test
	public void getLocalbyAcceptLanguage() {
		MockHttpServletRequest request = new MockHttpServletRequest();
		request.addHeader("accept-language", "en, jp;q=0.7, ko;q=0.8");
		//Locale cLocale = new Locale.Builder().setLanguage("en").setRegion("GB").build();
		Locale cLocale = new Locale("en");
		assertEquals(cLocale, LocaleUtil.getLocale(request));
	}
	
	@Test 
	public void testPara() {
		MockHttpServletRequest request = new MockHttpServletRequest();
		request.addParameter("key", "value");
		assertEquals("value", request.getParameter("key"));
	}
}
