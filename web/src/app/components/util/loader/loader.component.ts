import {Component} from '@angular/core';

@Component({
    selector: 'app-loader',
    standalone: true,
    imports: [],
    template: `
        <div class="card loader">
            <span class="spinner spinner-inline spinner-inline-custom"></span><span>Loading...</span>
        </div>
    `,
    styles: `
  .loader{
    padding: 1.25rem;
    width: 9.375rem;
  }
  .spinner-inline-custom{
    margin-right:0.625rem;
    }
  `
})
export class LoaderComponent {

}
