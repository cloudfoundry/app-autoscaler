package org.cloudfoundry.autoscaler.scheduler.entity;

import java.io.Serializable;
import java.sql.Array;
import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;

import org.apache.commons.lang3.ArrayUtils;
import org.hibernate.HibernateException;
import org.hibernate.engine.spi.SessionImplementor;
import org.hibernate.usertype.UserType;


/**
 * This is a user defined Type class. This class is created to handle the integer arrays, so as to 
 * be able to map them to PostgreSQL integer[].
 *
 */
public class IntArrayUserType implements UserType {
	protected static final int SQLTYPE = java.sql.Types.ARRAY;

	@Override
	public Object nullSafeGet(final ResultSet rs, final String[] names, final SessionImplementor sessionImplementor,
			final Object owner) throws HibernateException, SQLException {
		String columnName = names[0];
		Array array = rs.getArray(columnName);
		if (array == null) {
			return null;
		} else {
			Integer[] javaArray = (Integer[]) array.getArray();
			return ArrayUtils.toPrimitive(javaArray);
		}
	}

	@Override
	public void nullSafeSet(final PreparedStatement statement, final Object value, final int index,
			final SessionImplementor sessionImplementor) throws HibernateException, SQLException {
		Connection connection = statement.getConnection();

		if (value == null) {
			statement.setNull(index, SQLTYPE);
		} else {
			int[] castObject = (int[]) value;
			Integer[] integers = ArrayUtils.toObject(castObject);
			Array array = connection.createArrayOf("integer", integers);

			statement.setArray(index, array);
		}
	}

	@Override
	public Object assemble(final Serializable cached, final Object owner) throws HibernateException {
		return cached;
	}

	@Override
	public Object deepCopy(final Object o) throws HibernateException {
		return o == null ? null : ((int[]) o).clone();
	}

	@Override
	public Serializable disassemble(final Object o) throws HibernateException {
		return (Serializable) o;
	}

	@Override
	public boolean equals(final Object x, final Object y) throws HibernateException {
		return x == null ? y == null : x.equals(y);
	}

	@Override
	public int hashCode(final Object o) throws HibernateException {
		return o == null ? 0 : o.hashCode();
	}

	@Override
	public boolean isMutable() {
		return false;
	}

	@Override
	public Object replace(final Object original, final Object target, final Object owner) throws HibernateException {
		return original;
	}

	@Override
	public Class<int[]> returnedClass() {
		return int[].class;
	}

	@Override
	public int[] sqlTypes() {
		return new int[] { SQLTYPE };
	}
}