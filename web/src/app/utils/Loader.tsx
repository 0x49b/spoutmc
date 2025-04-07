import * as React from 'react';
import {Spinner} from "@patternfly/react-core";

const Loader: React.FunctionComponent = () => {


  return (
    <React.Fragment>
      <h2>
        Loading <Spinner isInline aria-label="spinner in a subheading"/>
      </h2>
    </React.Fragment>
  );
};

export {Loader};
