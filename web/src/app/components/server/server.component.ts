import {Component, OnInit} from '@angular/core';
import {HttpClient} from "@angular/common/http";

@Component({
  selector: 'app-server',
  standalone: true,
  imports: [],
  templateUrl: './server.component.html',
  styleUrl: './server.component.css'
})
export class ServerComponent implements OnInit {

  constructor(private http: HttpClient) {
  }

  ngOnInit() {

    this.http.get("localhost:3000/api/v1/container").subscribe(
      data => {
        console.log(data)
      }
    )
  }


}
