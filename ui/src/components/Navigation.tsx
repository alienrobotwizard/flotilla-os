import * as React from "react"
import { Link, NavLink } from "react-router-dom"
import {
  ButtonGroup,
  Navbar,
  NavbarDivider,
  NavbarGroup,
  Alignment,
  Classes,
} from "@blueprintjs/core"

const Navigation: React.FunctionComponent = () => (
  <Navbar fixedToTop className="bp3-dark">
    <NavbarGroup align={Alignment.LEFT}>
      <Link to="/templates" className="bp3-button bp3-minimal">
        Flotilla
      </Link>
      <NavbarDivider />
      <ButtonGroup className={Classes.MINIMAL}>
        <NavLink
          to="/templates"
          className={Classes.BUTTON}
          activeClassName={Classes.ACTIVE}
        >
          Templates
        </NavLink>
        <NavLink
          to="/runs"
          className={Classes.BUTTON}
          activeClassName={Classes.ACTIVE}
        >
          Runs
        </NavLink>
      </ButtonGroup>
    </NavbarGroup>
  </Navbar>
)

export default Navigation
