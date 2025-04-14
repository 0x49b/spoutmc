import * as React from 'react';
import '@patternfly/react-core/dist/styles/base.css';
import { BrowserRouter as Router } from 'react-router-dom';
import { AppLayout } from '@app/AppLayout/AppLayout';
import { AppRoutes } from '@app/routes';
import '@app/app.css';
import { Provider } from 'react-redux';
import { store } from '@app/store/store';
import { WebSocketProvider } from '@app/connection/WebSocketContext';
import { MqttProvider } from '@app/connection/MqttContext';


const App: React.FunctionComponent = () => (
  <Provider store={store}>
    <MqttProvider>
      <Router>
        <AppLayout>
          <AppRoutes />
        </AppLayout>
      </Router>
    </MqttProvider>
  </Provider>
);

export default App;
