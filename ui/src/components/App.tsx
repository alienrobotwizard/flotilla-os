import * as React from "react"
import { BrowserRouter, Route, Switch, Redirect } from "react-router-dom"

import Run from "./Run"
import Runs from "./Runs"
import Templates from "./Templates"
import Template from "./Template"
import Navigation from "./Navigation"
import { connect, ConnectedProps } from "react-redux"

const connector = connect()

class App extends React.Component<ConnectedProps<typeof connector>> {
  render() {
    return (
      <div className="flotilla-app-container bp3-dark">
        <BrowserRouter>
          <Navigation />
          <Switch>
            <Route exact path="/templates" component={Templates} />
            <Route path="/templates/:templateID" component={Template} />

            <Route exact path="/runs" component={Runs} />
            <Route path="/runs/:runID" component={Run} />
            <Redirect from="/" to="/templates" />
          </Switch>
        </BrowserRouter>
      </div>
    )
  }
}

export default connector(App)
