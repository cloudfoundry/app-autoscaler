package org.cloudfoundry.autoscaler.scheduler.util.error;

import java.io.Serial;
import org.springframework.beans.factory.BeanCreationException;

public class DataSourceConfigurationException extends BeanCreationException {
  /** */
  @Serial private static final long serialVersionUID = 6522875947154155552L;

  public DataSourceConfigurationException(String dataSourceName, String msg, Throwable t) {
    super(dataSourceName, msg, t);
  }
}
