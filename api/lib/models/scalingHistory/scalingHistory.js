'use strict';

module.exports = function(sequelize, DataTypes) {
  var ScalingHistory = sequelize.define('scalinghistory', {
    app_guid: {
      type: DataTypes.STRING,
      field: 'appid',
      primaryKey: true
    },
    timestamp: {
      type: DataTypes.BIGINT,
      field: 'timestamp',
      allowNull: false
    },
    scaling_type: {
      type: DataTypes.INTEGER,
      field: 'scalingtype',
      allowNull: false
    },
    status: {
      type: DataTypes.INTEGER,
      field: 'status',
      allowNull: false
    },
    old_instances: {
      type: DataTypes.INTEGER,
      field: 'oldinstances',
      allowNull: false
    },
    new_instances: {
      type: DataTypes.INTEGER,
      field: 'newinstances',
      allowNull: false
    },
    reason: {
      type: DataTypes.STRING,
      field: 'reason',
      allowNull: false
    },
    message: {
      type: DataTypes.STRING,
      field: 'message'
    },
    error: {
      type: DataTypes.STRING,
      field: 'error'
    }
  }, {
    timestamps: false,
    freezeTableName: true
  });
  return ScalingHistory;
};
