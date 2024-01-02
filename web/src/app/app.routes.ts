import {Routes} from '@angular/router';
import { BooksComponent } from './components/books/books.component';
import {ServerComponent} from "./components/server/server.component";
import {HomeComponent} from "./components/home/home.component";
import {ServerEditComponent} from "./components/server-edit/server-edit.component";

export const routes: Routes = [
  {
    path: 'books',
    component: BooksComponent
  },
  {
    path: 'server',
    component: ServerComponent
  },
  {
    path: 'server/edit/:serverId',
    component: ServerEditComponent,
  },
  {
    path: '',
    component: HomeComponent
  },
];
