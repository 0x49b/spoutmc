import {Routes} from '@angular/router';
import {BooksComponent} from './components/books/books.component';
import {ServerComponent} from "./components/server/server.component";
import {HomeComponent} from "./components/home/home.component";
import {ServerEditComponent} from "./components/server-edit/server-edit.component";
import {NewServerComponent} from "./components/server/new-server/new-server.component";
import {PlayerComponent} from "./components/player/player.component";
import {PlayerbanlistComponent} from "./components/player/banlist/playerbanlist.component";

export const routes: Routes = [
  {
    path: 'books',
    component: BooksComponent
  },
  {
    title: 'server',
    path: 'server',
    component: ServerComponent
  },
  {
    title: 'server-details',
    path: 'server/edit/:serverId',
    component: ServerEditComponent,
  },
  {
    title: 'new-server',
    path: 'server/new',
    component: NewServerComponent,
  },
  {
    title: 'player',
    path: 'player',
    component: PlayerComponent,
  },
  {
    title: 'player-banlist',
    path: 'banlist',
    component: PlayerbanlistComponent,
  },
  {
    title: 'dashboard',
    path: 'dashboard',
    component: HomeComponent,
  },
  {
    path: '',
    component: HomeComponent
  },
];
