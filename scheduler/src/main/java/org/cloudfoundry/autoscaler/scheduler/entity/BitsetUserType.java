package org.cloudfoundry.autoscaler.scheduler.entity;

import java.io.Serializable;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.ArrayList;
import java.util.List;

import org.hibernate.HibernateException;
import org.hibernate.engine.spi.SharedSessionContractImplementor;
import org.hibernate.usertype.UserType;

/**
 * This is a user defined Type class. This class is created to handle the integer arrays, so as to 
 * be able to map them to PostgreSQL integer[].
 *
 */
public class BitsetUserType implements UserType {
	protected static final int SQLTYPE = java.sql.Types.INTEGER;

	@Override
	public Object nullSafeGet(ResultSet rs, String[] names,
			SharedSessionContractImplementor sharedSessionContractImplementor, Object owner)
			throws HibernateException, SQLException {
		String columnName = names[0];
		int value = rs.getInt(columnName);

		List<Integer> javaArray = new ArrayList<>();
		if (value == 0) {
			return null;
		} else {
			for (int i = 0; (1L << i) < value; i++) {
				if ((value & (1L << i)) != 0) {
					javaArray.add(i + 1);
				}
			}
		}

		return javaArray.stream().mapToInt(i -> i).toArray();

	}

	@Override
	public void nullSafeSet(PreparedStatement statement, Object value, int index,
			SharedSessionContractImplementor sharedSessionContractImplementor) throws HibernateException, SQLException {
		if (value == null) {
			statement.setNull(index, SQLTYPE);
		} else {
			int[] castObject = (int[]) value;

			int bitset = 0;

			for (int aCastObject : castObject) {
				bitset |= 1 << (aCastObject - 1);
			}

			statement.setInt(index, bitset);
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
	public Class<Integer> returnedClass() {
		return int.class;
	}

	@Override
	public int[] sqlTypes() {
		return new int[] { SQLTYPE };
	}
}