'use strict';
module.exports = function(sequelize, DataTypes) {
  var service_instance = sequelize.define('service_instance', {
    serviceInstanceId: {
      type: DataTypes.STRING,
      primaryKey: true,
      field: 'service_instance_id',
      allowNull: false
    },
    orgId: {
      type: DataTypes.STRING,
      field: 'org_id',
      allowNull: false
    },
    spaceId: {
      type: DataTypes.STRING,
      field: 'space_id',
      allowNull: false
    },
  }, {
    classMethods: {
      associate: function(models) {
        service_instance.hasMany(models.binding, { foreignKey: 'service_instance_id' });
      }
    },
    timestamps: false,
    freezeTableName: true
  });
  return service_instance;
};