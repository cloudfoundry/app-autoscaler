package org.cloudfoundry.autoscaler.common.util;

import java.util.HashSet;
import java.util.Locale;
import java.util.Set;
import java.util.regex.Pattern;

import javax.servlet.http.HttpServletRequest;

public class LocaleUtil {
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
	private static final Pattern languagePattern = Pattern.compile("([a-z]{2}[[-[A-Z]{2}]?[;q=[0.[0-9]]|1]?])*,");
	private static final Pattern qPattern = Pattern.compile("[;q=[0.[0-9]]|1]");
	private static final Pattern localePattern = Pattern.compile("[a-z]{2}-[A-Z]{2}");

	public static Locale getLocale(HttpServletRequest httpServletRequest) {

		Locale locale = httpServletRequest.getLocale();
		if (locale.toString().isEmpty() || locale.toString().equalsIgnoreCase("null")) {
			try {
				String acceptLanguageHeader = httpServletRequest.getHeader("Accept-Language");

				String[] acceptLanguages = languagePattern.split(acceptLanguageHeader);
				for (String acceptLanguage : acceptLanguages) {
					acceptLanguage = qPattern.matcher(acceptLanguage).replaceAll("").trim();
					if (localePattern.matcher(acceptLanguage).matches()) {
						String[] languageAndCountry = acceptLanguage.split("-");
						String language = languageAndCountry[0].trim();
						String country = languageAndCountry[1].trim();
						if (supportedLanguage.contains(language)) {
							locale = new Locale(language, country, "");
							break;
						}
					} else {
						if (supportedLanguage.contains(acceptLanguage)) {
							locale = new Locale(acceptLanguage);
							break;
						}
					}
				}
			} catch (Exception e) {
				return DEFAULT_LOCALE;
			}
		}
		return locale;
	}
}
