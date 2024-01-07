import {Component} from '@angular/core';
import {FormBuilder, FormControl, ReactiveFormsModule, Validators} from "@angular/forms";
import {Router} from "@angular/router";
import {RestService} from "../../../services/rest/rest.service";

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

  constructor(private formBuilder: FormBuilder,
              private router: Router,
              private restService: RestService) {
  }


  onSubmit() {
    console.log(this.newServerForm)
  }

  createNewServer(name: string) {
    this.restService.createNewServer(name).subscribe(data => {
      console.log(data)
    })
  }

  cancel() {
    this.newServerForm.reset()
    this.router.navigateByUrl('/server')
  }
}
