import {Component, OnInit} from '@angular/core';
import {HttpClient} from "@angular/common/http";




export interface MCServer {
  name: string[],
  state: string
}

@Component({
  selector: 'app-server',
  standalone: true,
  imports: [],
  templateUrl: './server.component.html',
  styleUrl: './server.component.css'
})
export class ServerComponent implements OnInit {

  displayedColumns: string[] = ['name', 'state']
  dataSource: MCServer = []


  constructor(private http: HttpClient) {
  }

  ngOnInit() {

  this.http.get<MCServer>("http://localhost:3000/api/v1/container").subscribe(
      data => {
        console.log(data)
        this.dataSource = data
      }
    )
  }


}
