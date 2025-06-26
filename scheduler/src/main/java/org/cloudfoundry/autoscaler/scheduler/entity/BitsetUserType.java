package org.cloudfoundry.autoscaler.scheduler.entity;

import java.io.Serializable;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import org.hibernate.HibernateException;
import org.hibernate.engine.spi.SharedSessionContractImplementor;
import org.hibernate.usertype.UserType;

/**
 * This is a user defined Type class. This class is created to handle the integer arrays, so as to
 * be able to map them to PostgreSQL integer[].
 */
public class BitsetUserType implements UserType<int[]> {
  protected static final int SQLTYPE = java.sql.Types.INTEGER;

  @Override
  public int[] nullSafeGet(
      ResultSet resultSet,
      int columnIndex,
      SharedSessionContractImplementor sharedSessionContractImplementor,
      Object o)
      throws SQLException {
    int value = resultSet.getInt(columnIndex);

    List<Integer> javaArray = new ArrayList<>();
    if (value == 0) {
      return new int[0];
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
  public void nullSafeSet(
      PreparedStatement statement,
      int[] ints,
      int index,
      SharedSessionContractImplementor sharedSessionContractImplementor)
      throws HibernateException, SQLException {
    if (ints == null) {
      statement.setNull(index, SQLTYPE);
    } else {

      int bitset = 0;

      for (int i : ints) {
        bitset |= 1 << (i - 1);
      }

      statement.setInt(index, bitset);
    }
  }

  @Override
  public int[] assemble(final Serializable cached, final Object owner) throws HibernateException {
    return (int[]) cached;
  }

  @Override
  public int[] deepCopy(int[] ints) {
    return ints == null ? null : ints.clone();
  }

  @Override
  public Serializable disassemble(int[] ints) {
    return ints;
  }

  @Override
  public boolean equals(int[] ints, int[] j1) {
    return Arrays.equals(ints, j1);
  }

  @Override
  public int hashCode(int[] ints) {
    return Arrays.hashCode(ints);
  }

  @Override
  public boolean isMutable() {
    return false;
  }

  @Override
  public int[] replace(int[] ints, int[] j1, Object o) {
    return ints;
  }

  @Override
  public Class<int[]> returnedClass() {
    return int[].class;
  }

  @Override
  public int getSqlType() {
    return SQLTYPE;
  }
}
