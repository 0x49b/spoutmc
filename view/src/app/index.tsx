import * as React from 'react';
import '@patternfly/react-core/dist/styles/base.css';
import {BrowserRouter as Router} from 'react-router-dom';
import {AppLayout} from '@app/AppLayout/AppLayout';
import {AppRoutes} from '@app/routes';
import '@app/app.css';
import '@patternfly/react-core/dist/styles/base.css';
import {Provider} from "react-redux";
import {store} from "@app/store/store";

const App: React.FunctionComponent = () => (
  <Provider store={store}>
    <Router>
      <AppLayout>
        <AppRoutes/>
      </AppLayout>
    </Router>
  </Provider>
);

export default App;
