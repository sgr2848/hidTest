import "./App.css";
import { useEffect, useState } from "react";
import FobComponent from "./assets/component/Fob";
import { LoginButton } from "./assets/component/LoginButton";
import { Dashboard } from "./assets/component/Dashboard";
import axios from "axios";
function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    const checkAuthentication = async () => {
      try {
        const response = await axios.get(
          "http://localhost:8000/auth/saml/is_authenticated",
          {
            withCredentials: true,
            headers: {
              Authorization: "Bearer " + localStorage.getItem("token"),
            },
          }
        );
        setIsAuthenticated(response.data.isAuthenticated);
      } catch (error) {
        console.error("Failed to check authentication status:", error);
      }
    };

    checkAuthentication();
  }, []);
  return (
    <>
      {isAuthenticated ? <Dashboard /> : <LoginButton />}
      <FobComponent />
    </>
  );
}

export default App;
