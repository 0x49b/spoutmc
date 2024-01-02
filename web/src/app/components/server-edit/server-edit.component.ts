import {Component, inject} from '@angular/core';
import {ActivatedRoute} from "@angular/router";

@Component({
  selector: 'app-server-edit',
  standalone: true,
  imports: [],
  templateUrl: './server-edit.component.html',
  styleUrl: './server-edit.component.css'
})
export class ServerEditComponent {

  privateActivatedRoute = inject(ActivatedRoute)
  serverId = ""

  constructor() {
    this.privateActivatedRoute = inject(ActivatedRoute)
    this.serverId = this.privateActivatedRoute.snapshot.params['serverId'];
  }


}
