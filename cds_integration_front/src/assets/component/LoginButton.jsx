// import axios from "axios";

export function LoginButton() {
  const handleLogin = async () => {
    //   const response = await axios.get("http://localhost:8000/saml/login");
    //   const samlLoginUrl = response.data.url;
    window.location.href = "http://localhost:8000/saml/login"; // Redirect to SAML IdP
  };

  return <button onClick={handleLogin}>Login with SAML</button>;
}
