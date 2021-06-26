const user = {
    user: 'fluentd',
    pwd: 'fluentdPassword',
    roles: [{
      role: 'readWrite',
      db: 'logViewer'
    }]
  };
db.createUser(user);