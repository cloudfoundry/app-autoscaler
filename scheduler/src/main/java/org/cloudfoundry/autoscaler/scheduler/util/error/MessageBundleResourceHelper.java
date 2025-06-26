package org.cloudfoundry.autoscaler.scheduler.util.error;

import java.util.Locale;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.context.MessageSource;
import org.springframework.context.i18n.LocaleContextHolder;
import org.springframework.stereotype.Component;

/**
 * Helper class for looking up message bundle resources It can be used to lookup strings from
 * messages.properties
 */
@Component
public class MessageBundleResourceHelper {

  @Autowired
  @Qualifier("messageSource")
  private MessageSource messageSource;

  /**
   * Lookup a message resource for the specified key
   *
   * @param key - the key
   * @param defaultMessage - a default to return if the key is not found
   * @param arguments - any arguments needed by the message resource
   * @return - the located message or defaultMessage if none is found
   */
  private String lookupMessageWithDefault(String key, String defaultMessage, Object... arguments) {

    Locale locale = LocaleContextHolder.getLocale();

    return messageSource.getMessage(key, arguments, defaultMessage, locale);
  }

  /**
   * Lookup a message resource for the specified key, using <code>"["+key+"]"</code> as default
   * message.
   */
  public String lookupMessage(String key, Object... arguments) {
    return lookupMessageWithDefault(key, "[" + key + "]", arguments);
  }
}
