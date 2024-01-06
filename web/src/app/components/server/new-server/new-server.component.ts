import {Component} from '@angular/core';
import {FormBuilder, FormControl, ReactiveFormsModule, Validators} from "@angular/forms";
import {HttpHeaders} from "@angular/common/http";
import {Router} from "@angular/router";

@Component({
  selector: 'app-new-server',
  standalone: true,
  imports: [
    ReactiveFormsModule
  ],
  templateUrl: './new-server.component.html',
  styleUrl: './new-server.component.css'
})
export class NewServerComponent {

  newServerForm = this.formBuilder.group({
    name: new FormControl('', [Validators.required, Validators.minLength(5)]),
    proxy: false,
    lobby: false
  })

  constructor(private formBuilder: FormBuilder, private router: Router) {
  }


  onSubmit() {
    console.log(this.newServerForm)
  }

  createNewServer(name: string) {
    const headers = new HttpHeaders({'Content-Type': 'application/json'});
    /* this.http.post<any>("http://localhost:3000/api/v1/container/create",
       JSON.stringify({servername: name}),
       {headers}
     ).subscribe(
       data => {
         console.log(data)
       })*/
  }

  cancel() {
    this.newServerForm.reset()
    this.router.navigateByUrl('/server')
  }
}
