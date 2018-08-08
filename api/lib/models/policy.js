'use strict';

module.exports = function(sequelize, DataTypes) {
  var Policy = sequelize.define('policy_json', {
    app_id: {
      type: DataTypes.STRING,
      field: 'app_id',
      primaryKey: true
    },
    policy_json: {
      type: DataTypes.JSON,
      field: 'policy_json',
      allowNull: false
    },
    guid: {
      type: DataTypes.STRING,
      field: 'guid',
      allowNull: false
    }
  },{
    timestamps: false,
    freezeTableName: true
  });

  return Policy;
};
