import {Routes} from '@angular/router';
import {BooksComponent} from "./modules/books/books.component";
import {HomeComponent} from "./modules/home/home.component";
import {ServerComponent} from "./modules/server/server.component";

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
    path: '',
    component: HomeComponent
  },
];
