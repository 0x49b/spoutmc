const path = require('path');
module.exports = {
  stylePaths: [
    path.resolve(__dirname, 'src'),
    path.resolve(__dirname, 'node_modules/patternfly'),
    path.resolve(__dirname, 'node_modules/@patternfly/patternfly'),
    path.resolve(__dirname, 'node_modules/@patternfly/react-styles/css'),
    path.resolve(__dirname, 'node_modules/@patternfly/react-internal/dist/styles/base.css'),
    path.resolve(__dirname, 'node_modules/@patternfly/react-internal/dist/esm/@patternfly/patternfly'),
    path.resolve(__dirname, 'node_modules/@patternfly/react-internal/node_modules/@patternfly/react-styles/css'),
    path.resolve(__dirname, 'node_modules/@patternfly/react-table/node_modules/@patternfly/react-styles/css'),
    path.resolve(__dirname, 'node_modules/@patternfly/react-inline-edit-extension/node_modules/@patternfly/react-styles/css')
  ]
}
