'use strict';
module.exports = function(sequelize, DataTypes) {
  var binding = sequelize.define('binding', {
    bindingId: {
      type: DataTypes.STRING,
      primaryKey: true,
      field: 'binding_id',
      allowNull: false
    },
    appId: {
      type: DataTypes.STRING,
      field: 'app_id',
      allowNull: false
    },
    serviceInstanceId: {
      type: DataTypes.STRING,
      field: 'service_instance_id',
      allowNull: false
    }
  }, {
    classMethods: {
      associate: function(models) {
        binding.belongsTo(models.service_instance, { foreignKey: 'service_instance_id',onDelete: "RESTRICT",onUpdate:"CASCADE"});
      },

    },
    timestamps: true,
    createdAt: 'created_at',
    updatedAt: false,
    freezeTableName: true
  });
  return binding;
};