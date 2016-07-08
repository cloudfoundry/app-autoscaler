package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.Comparator;

public class DateTimeComparator implements Comparator<SpecificDateScheduleDateTime> {

	public int compare(SpecificDateScheduleDateTime first, SpecificDateScheduleDateTime second) {
		if ((first == null) && (second == null))
			return 0;
		if (first == null)
			throw new RuntimeException("first object is null in DateTimeComparator");
		if (second == null)
			throw new RuntimeException("second object is null in DateTimeComparator");

		Long firstValue = first.getStartDateTime();
		Long secondValue = second.getStartDateTime();

		if (firstValue == null)
			throw new RuntimeException("firstValue is null in DateTimeComparator");
		if (secondValue == null)
			throw new RuntimeException("secondValue is null in DateTimeComparator");

		if (firstValue > secondValue) {
			return 1;
		} else if (firstValue == secondValue) {
			return 0;
		} else {
			return -1;
		}
	}
}
